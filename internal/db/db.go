package db

import (
	"fmt"
	"log"
	"time"

	"attendance-collector/internal/config"
	"attendance-collector/internal/models"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Open connects to the configured database and returns a *gorm.DB.
func Open(c *config.Config) (*gorm.DB, error) {
	gcfg := &gorm.Config{
		Logger:  logger.Default.LogMode(logger.Silent),
		NowFunc: func() time.Time { return time.Now().Local() },
	}

	var (
		dialector gorm.Dialector
		desc      string
	)

	switch c.DBDriver {
	case "mysql":
		dialector = mysql.Open(c.MySQLDSN)
		desc = "mysql"
	case "sqlite", "":
		dialector = sqlite.Open(c.SQLitePath)
		desc = "sqlite:" + c.SQLitePath
	default:
		return nil, fmt.Errorf("unsupported DB_DRIVER %q (use sqlite or mysql)", c.DBDriver)
	}

	gdb, err := gorm.Open(dialector, gcfg)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", desc, err)
	}

	if c.AutoMigrate {
		if err := gdb.AutoMigrate(&models.Device{}, &models.Employee{}, &models.AttendanceLog{}, &models.Attendance{}); err != nil {
			return nil, fmt.Errorf("automigrate: %w", err)
		}
	}

	log.Printf("database ready (%s)", desc)
	return gdb, nil
}
