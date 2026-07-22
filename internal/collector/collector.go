// Package collector ties the device client to the database: it syncs employees
// and pulls attendance punches, storing everything without duplicates.
package collector

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"attendance-collector/internal/config"
	"attendance-collector/internal/models"
	"attendance-collector/internal/zk"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Collector struct {
	db    *gorm.DB
	cfg   *config.Config
	locks sync.Map // map[string]*sync.Mutex
}

func New(db *gorm.DB, cfg *config.Config) *Collector {
	return &Collector{db: db, cfg: cfg}
}

func (c *Collector) deviceMutex(key string) *sync.Mutex {
	v, _ := c.locks.LoadOrStore(key, &sync.Mutex{})
	return v.(*sync.Mutex)
}

// device returns the DB row for a configured device, creating/refreshing it.
func (c *Collector) device(dc config.DeviceConfig) (*models.Device, error) {
	var dev models.Device
	err := c.db.Where("ip_address = ? AND port = ?", dc.IP, dc.Port).First(&dev).Error
	if err == gorm.ErrRecordNotFound {
		dir := dc.Direction
		loc := dc.Direction
		dev = models.Device{
			Name:      dc.Name,
			IPAddress: dc.IP,
			Port:      dc.Port,
			Direction: &dir,
			Location:  &loc,
			IsActive:  true,
		}
		if err := c.db.Create(&dev).Error; err != nil {
			return nil, err
		}
		return &dev, nil
	}
	if err != nil {
		return nil, err
	}
	return &dev, nil
}

func (c *Collector) markDevice(dev *models.Device, status string, pullErr error, pulled, synced bool) {
	now := time.Now()
	updates := map[string]any{"last_status": status}
	if pullErr != nil {
		msg := pullErr.Error()
		updates["last_error"] = msg
	} else {
		updates["last_error"] = nil
	}
	if pulled {
		updates["last_pulled_at"] = now
	}
	if synced {
		updates["last_synced_at"] = now
	}
	c.db.Model(&models.Device{}).Where("id = ?", dev.ID).Updates(updates)
}

// SyncEmployees pulls the enrolled users from a device and upserts them.
func (c *Collector) SyncEmployees(dc config.DeviceConfig) error {
	mu := c.deviceMutex(dc.Address())
	if !mu.TryLock() {
		log.Printf("[%s] employee sync skipped: previous operation on device is still running", dc.Name)
		return nil
	}
	defer mu.Unlock()

	dev, err := c.device(dc)
	if err != nil {
		return err
	}

	users, err := zk.New(dc, c.cfg).Users()
	if err != nil {
		c.markDevice(dev, "error", err, false, false)
		return err
	}

	count := 0
	for _, u := range users {
		pin := u.UserPin
		if pin == "0" || pin == "" {
			continue
		}
		name := strings.TrimSpace(u.Name)
		if name == "" {
			name = "PIN " + pin
		}
		var card *string
		if strings.TrimSpace(u.Card) != "" {
			cardVal := u.Card
			card = &cardVal
		}

		emp := models.Employee{
			DeviceUserPin: pin,
			Name:          name,
			CardNo:        card,
			Role:          uint8(u.Privilege),
			IsActive:      true,
		}
		// Insert, or update name/card/role on conflict of the unique PIN.
		if err := c.db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "device_user_pin"}},
			DoUpdates: clause.AssignmentColumns([]string{"name", "card_no", "role", "updated_at"}),
		}).Create(&emp).Error; err != nil {
			return err
		}
		count++
	}

	c.markDevice(dev, "ok", nil, false, true)
	log.Printf("[%s] synced %d employees", dc.Name, count)
	return nil
}

// PullAttendance reads all punches from a device and stores the new ones.
func (c *Collector) PullAttendance(dc config.DeviceConfig) error {
	mu := c.deviceMutex(dc.Address())
	if !mu.TryLock() {
		log.Printf("[%s] pull skipped: previous operation on device is still running", dc.Name)
		return nil
	}
	defer mu.Unlock()

	dev, err := c.device(dc)
	if err != nil {
		return err
	}

	logs, err := zk.New(dc, c.cfg).Attendance()
	if err != nil {
		c.markDevice(dev, "error", err, false, false)
		return err
	}

	newCount, dupCount, err := c.storePunches(dev, logs)
	if err != nil {
		c.markDevice(dev, "error", err, false, false)
		return err
	}

	c.markDevice(dev, "ok", nil, true, false)
	log.Printf("[%s] read %d, new %d, duplicates %d", dc.Name, len(logs), newCount, dupCount)
	return nil
}

// storePunches resolves employees, builds daily aggregated attendance records,
// and upserts them into the attendances table. Returns (new, duplicates).
func (c *Collector) storePunches(dev *models.Device, logs []zk.Punch) (int, int, error) {
	if len(logs) == 0 {
		return 0, 0, nil
	}

	// 1. Resolve Employees
	pins := map[string]struct{}{}
	for _, l := range logs {
		pins[l.UserPin] = struct{}{}
	}
	empByPin, err := c.resolveEmployees(pins)
	if err != nil {
		return 0, 0, err
	}

	// 2. Build unique raw logs keyed by dedupe hash
	rowByHash := map[string]models.AttendanceLog{}
	for _, l := range logs {
		pin := l.UserPin
		if pin == "" {
			continue
		}
		empID := empByPin[pin]
		hash := models.MakeDedupeHash(&dev.ID, pin, l.Time)
		rowByHash[hash] = models.AttendanceLog{
			DeviceID:   &dev.ID,
			EmployeeID: &empID,
			UserPin:    pin,
			PunchedAt:  l.Time,
			State:      uint8(l.State),
			Type:       uint8(l.Type),
			DedupeHash: hash,
			CreatedAt:  time.Now(),
		}
	}

	// 3. Find which hashes already exist in attendance_logs
	hashes := make([]string, 0, len(rowByHash))
	for h := range rowByHash {
		hashes = append(hashes, h)
	}
	existingHashes := map[string]struct{}{}
	for _, chunk := range chunkStrings(hashes, 500) {
		var found []string
		if err := c.db.Model(&models.AttendanceLog{}).
			Where("dedupe_hash IN ?", chunk).
			Pluck("dedupe_hash", &found).Error; err != nil {
			return 0, 0, err
		}
		for _, h := range found {
			existingHashes[h] = struct{}{}
		}
	}

	// 4. Filter only NEW raw logs
	newLogs := make([]models.AttendanceLog, 0, len(rowByHash))
	for h, row := range rowByHash {
		if _, dup := existingHashes[h]; !dup {
			newLogs = append(newLogs, row)
		}
	}

	if len(newLogs) == 0 {
		return 0, len(rowByHash), nil // all were duplicates
	}

	// 5. Insert new raw logs into attendance_logs in batches
	for _, batch := range chunkRawRows(newLogs, 500) {
		if err := c.db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "dedupe_hash"}},
			DoNothing: true,
		}).Create(&batch).Error; err != nil {
			return 0, 0, err
		}
	}

	// 6. Aggregate these NEW logs into daily attendances
	type empDateKey struct {
		EmpID    uint
		WorkDate time.Time
	}
	type dailyAgg struct {
		CheckIn    time.Time
		CheckOut   time.Time
		PunchCount int
	}

	groups := map[empDateKey]*dailyAgg{}
	dates := map[time.Time]struct{}{}

	for _, l := range newLogs {
		y, m, d := l.PunchedAt.Date()
		workDate := time.Date(y, m, d, 0, 0, 0, 0, l.PunchedAt.Location())
		key := empDateKey{EmpID: *l.EmployeeID, WorkDate: workDate}
		dates[workDate] = struct{}{}

		if agg, exists := groups[key]; exists {
			if l.PunchedAt.Before(agg.CheckIn) {
				agg.CheckIn = l.PunchedAt
			}
			if l.PunchedAt.After(agg.CheckOut) {
				agg.CheckOut = l.PunchedAt
			}
			agg.PunchCount++
		} else {
			groups[key] = &dailyAgg{
				CheckIn:    l.PunchedAt,
				CheckOut:   l.PunchedAt,
				PunchCount: 1,
			}
		}
	}

	var dateList []time.Time
	for d := range dates {
		dateList = append(dateList, d)
	}

	// Fetch existing records for these dates to merge check-in/out times
	var existingRows []models.Attendance
	if len(dateList) > 0 {
		if err := c.db.Where("work_date IN ?", dateList).Find(&existingRows).Error; err != nil {
			return 0, 0, err
		}
	}

	existingMap := make(map[empDateKey]models.Attendance)
	for _, row := range existingRows {
		key := empDateKey{EmpID: row.EmployeeID, WorkDate: row.WorkDate}
		existingMap[key] = row
	}

	now := time.Now()
	var upserts []models.Attendance

	for key, agg := range groups {
		ci := agg.CheckIn
		co := agg.CheckOut
		punchCount := agg.PunchCount

		if existing, hasExisting := existingMap[key]; hasExisting {
			if existing.CheckIn != nil && existing.CheckIn.Before(ci) {
				ci = *existing.CheckIn
			}
			if existing.CheckOut != nil && existing.CheckOut.After(co) {
				co = *existing.CheckOut
			}
			if existing.CheckIn != nil && existing.CheckIn.After(co) {
				co = *existing.CheckIn
			}
			punchCount += existing.PunchCount // Safe to ADD because we only process strictly new punches
		}

		var checkOutPtr *time.Time
		if ci != co {
			checkOutPtr = &co
		}

		workedMinutes := 0
		if checkOutPtr != nil {
			workedMinutes = int(checkOutPtr.Sub(ci).Minutes())
		}

		upserts = append(upserts, models.Attendance{
			EmployeeID:    key.EmpID,
			WorkDate:      key.WorkDate,
			CheckIn:       &ci,
			CheckOut:      checkOutPtr,
			WorkedMinutes: workedMinutes,
			BreakMinutes:  0,
			PunchCount:    punchCount,
			Status:        "Present",
			CreatedAt:     now,
			UpdatedAt:     now,
		})
	}

	// Upsert daily records in batches
	for _, batch := range chunkRows(upserts, 500) {
		if err := c.db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "employee_id"}, {Name: "work_date"}},
			DoUpdates: clause.AssignmentColumns([]string{"check_in", "check_out", "worked_minutes", "punch_count", "updated_at"}),
		}).Create(&batch).Error; err != nil {
			return 0, 0, err
		}
	}

	return len(newLogs), len(rowByHash) - len(newLogs), nil
}

// resolveEmployees maps every PIN to an employee id, creating placeholders for
// PINs not yet enrolled (so no punch is ever dropped).
func (c *Collector) resolveEmployees(pins map[string]struct{}) (map[string]uint, error) {
	list := make([]string, 0, len(pins))
	for p := range pins {
		list = append(list, p)
	}

	out := map[string]uint{}
	for _, chunk := range chunkStrings(list, 500) {
		var emps []models.Employee
		if err := c.db.Where("device_user_pin IN ?", chunk).Find(&emps).Error; err != nil {
			return nil, err
		}
		for _, e := range emps {
			out[e.DeviceUserPin] = e.ID
		}
	}

	for _, pin := range list {
		if _, ok := out[pin]; ok {
			continue
		}
		emp := models.Employee{
			DeviceUserPin: pin,
			Name:          "Unknown (PIN " + pin + ")",
			IsActive:      true,
		}
		if err := c.db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "device_user_pin"}},
			DoNothing: true,
		}).Create(&emp).Error; err != nil {
			return nil, err
		}
		if emp.ID == 0 {
			// Conflict path: fetch the existing id.
			if err := c.db.Where("device_user_pin = ?", pin).First(&emp).Error; err != nil {
				return nil, err
			}
		}
		out[pin] = emp.ID
	}
	return out, nil
}

// ClearDevice is permanently disabled to ensure hardware logs are never cleared.
func (c *Collector) ClearDevice(dc config.DeviceConfig) error {
	return fmt.Errorf("[%s] device log clearing is permanently disabled to ensure hardware data safety", dc.Name)
}

func chunkStrings(s []string, size int) [][]string {
	var out [][]string
	for i := 0; i < len(s); i += size {
		end := i + size
		if end > len(s) {
			end = len(s)
		}
		out = append(out, s[i:end])
	}
	return out
}

func chunkRows(s []models.Attendance, size int) [][]models.Attendance {
	var out [][]models.Attendance
	for i := 0; i < len(s); i += size {
		end := i + size
		if end > len(s) {
			end = len(s)
		}
		out = append(out, s[i:end])
	}
	return out
}

func chunkRawRows(s []models.AttendanceLog, size int) [][]models.AttendanceLog {
	var out [][]models.AttendanceLog
	for i := 0; i < len(s); i += size {
		end := i + size
		if end > len(s) {
			end = len(s)
		}
		out = append(out, s[i:end])
	}
	return out
}
