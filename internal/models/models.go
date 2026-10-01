// Package models defines the database tables. Column and table names mirror the
// existing Laravel attendance schema so this collector can write into the very
// same database (SQLite or MySQL) that the dashboard/reports read from.
package models

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

// Direction constants for AttendanceLog matching Laravel schema:
// 0 = Unknown/Undirected, 1 = IN (Entry), 2 = OUT (Exit)
const (
	DirectionUnknown uint8 = 0
	DirectionIn      uint8 = 1 // Entry / In
	DirectionOut     uint8 = 2 // Exit / Out
)

// Device is a physical ZKTeco unit we poll.
type Device struct {
	ID           uint       `gorm:"primaryKey"`
	Name         string     `gorm:"column:name;type:varchar(191)"`
	IPAddress    string     `gorm:"column:ip_address;type:varchar(191)"`
	Port         uint       `gorm:"column:port;default:4370"`
	Location     *string    `gorm:"column:location;type:varchar(191)"`
	Direction    *string    `gorm:"column:direction;type:varchar(191)"` // "in" | "out" | nil
	IsActive     bool       `gorm:"column:is_active;default:true"`
	LastPulledAt *time.Time `gorm:"column:last_pulled_at"`
	LastSyncedAt *time.Time `gorm:"column:last_synced_at"`
	LastStatus   *string    `gorm:"column:last_status;type:varchar(191)"`
	LastError    *string    `gorm:"column:last_error;type:text"`
	CreatedAt    time.Time  `gorm:"column:created_at"`
	UpdatedAt    time.Time  `gorm:"column:updated_at"`
}

func (Device) TableName() string { return "devices" }

// Employee is a person enrolled on a device (identified by the device PIN).
type Employee struct {
	ID            uint      `gorm:"primaryKey"`
	DeviceUserPin string    `gorm:"column:device_user_pin;uniqueIndex;type:varchar(191)"`
	Name          string    `gorm:"column:name;type:varchar(191)"`
	CardNo        *string   `gorm:"column:card_no;type:varchar(191)"`
	Department    *string   `gorm:"column:department;type:varchar(191)"`
	Role          uint8     `gorm:"column:role;default:0"`
	IsActive      bool      `gorm:"column:is_active;default:true"`
	CreatedAt     time.Time `gorm:"column:created_at"`
	UpdatedAt     time.Time `gorm:"column:updated_at"`
}

func (Employee) TableName() string { return "employees" }

// Attendance is a daily aggregated attendance record.
type Attendance struct {
	ID            uint       `gorm:"primaryKey"`
	EmployeeID    uint       `gorm:"column:employee_id;uniqueIndex:idx_emp_date"`
	WorkDate      time.Time  `gorm:"column:work_date;type:date;uniqueIndex:idx_emp_date"`
	CheckIn       *time.Time `gorm:"column:check_in"`
	CheckOut      *time.Time `gorm:"column:check_out"`
	WorkedMinutes int        `gorm:"column:worked_minutes;default:0"`
	BreakMinutes  int        `gorm:"column:break_minutes;default:0"`
	PunchCount    int        `gorm:"column:punch_count;default:0"`
	Status        string     `gorm:"column:status;type:varchar(191)"`
	CreatedAt     time.Time  `gorm:"column:created_at"`
	UpdatedAt     time.Time  `gorm:"column:updated_at"`
}

func (Attendance) TableName() string { return "attendances" }

// AttendanceLog is one raw punch, deduplicated by DedupeHash.
type AttendanceLog struct {
	ID           uint      `gorm:"primaryKey"`
	DeviceID     *uint     `gorm:"column:device_id"`
	EmployeeID   *uint     `gorm:"column:employee_id"`
	UserPin      string    `gorm:"column:user_pin;type:varchar(191)"`
	PunchedAt    time.Time `gorm:"column:punched_at"`
	State        uint8     `gorm:"column:state;default:0"` // verify mode (fp/card/pw)
	Type         uint8     `gorm:"column:type;default:0"`  // punch type (0 in, 1 out, ...)
	Direction    uint8     `gorm:"column:direction;default:0"` // 0: unknown, 1: in (entry), 2: out (exit)
	IsReconciled bool      `gorm:"column:is_reconciled;default:false"`
	DedupeHash   string    `gorm:"column:dedupe_hash;uniqueIndex;type:varchar(64)"`
	CreatedAt    time.Time `gorm:"column:created_at"`
}

func (AttendanceLog) TableName() string { return "attendance_logs" }

// ResolveDirection converts device direction string ("in", "out") or punch type into
// the integer direction expected by the attendance schema:
// 1 = IN (Entry), 2 = OUT (Exit), 0 = Unknown
func ResolveDirection(devDir *string, punchType uint8) uint8 {
	if devDir != nil {
		switch strings.ToLower(strings.TrimSpace(*devDir)) {
		case "in", "entry":
			return DirectionIn
		case "out", "exit":
			return DirectionOut
		}
	}
	switch punchType {
	case 0:
		return DirectionIn
	case 1:
		return DirectionOut
	default:
		return DirectionUnknown
	}
}

// MakeDedupeHash matches the Laravel implementation exactly:
//	sha256( "<deviceID>|<pin>|<Y-m-d H:i:s>" )
func MakeDedupeHash(deviceID *uint, pin string, punchedAt time.Time) string {
	id := ""
	if deviceID != nil {
		id = fmt.Sprintf("%d", *deviceID)
	}
	raw := fmt.Sprintf("%s|%s|%s", id, pin, punchedAt.Format("2006-01-02 15:04:05"))
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

