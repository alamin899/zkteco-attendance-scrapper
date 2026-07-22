// Package config loads all runtime settings from environment variables (and an
// optional .env file). It is intentionally small: two devices (an IN and an OUT
// unit, matching the physical setup), a database target, and polling knobs.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// DeviceConfig describes one physical ZKTeco unit.
type DeviceConfig struct {
	Name      string
	IP        string
	Port      uint
	Direction string // "in" | "out" | ""
}

// Address returns "ip:port" for the zkteco client.
func (d DeviceConfig) Address() string {
	return fmt.Sprintf("%s:%d", d.IP, d.Port)
}

// Config is the fully-resolved application configuration.
type Config struct {
	// Database
	DBDriver string // "sqlite" | "mysql"
	SQLitePath string
	MySQLDSN   string

	// Devices (only those with a non-empty IP are kept)
	Devices []DeviceConfig

	// ZK client behaviour
	Timeout        time.Duration
	Password       string
	Retries        int
	RetryDelay     time.Duration
	StrictChecksum bool

	// Scheduling (used by the `run` daemon)
	PollInterval time.Duration
	Stagger      time.Duration

	AutoMigrate bool
}

// Load reads configuration from the environment, applying sensible defaults.
func Load() (*Config, error) {
	// .env is optional; ignore a missing file.
	_ = godotenv.Load()

	dbDriver := strings.ToLower(getEnv("DB_CONNECTION", getEnv("DB_DRIVER", "sqlite")))
	dbPath := getEnv("DB_PATH", getEnv("DB_DATABASE", "attendance.db"))

	c := &Config{
		DBDriver:       dbDriver,
		SQLitePath:     dbPath,
		Timeout:        time.Duration(getEnvInt("ZK_TIMEOUT", 30)) * time.Second,
		Password:       getEnv("ZK_PASSWORD", "0"),
		Retries:        getEnvInt("ZK_RETRIES", 2),
		RetryDelay:     time.Duration(getEnvInt("ZK_RETRY_DELAY", 2)) * time.Second,
		StrictChecksum: getEnvBool("ZK_STRICT_CHECKSUM", false),
		PollInterval:   time.Duration(getEnvInt("POLL_INTERVAL_MINUTES", 10)) * time.Minute,
		Stagger:        time.Duration(getEnvInt("STAGGER_MINUTES", 5)) * time.Minute,
		AutoMigrate:    getEnvBool("AUTO_MIGRATE", true),
	}

	if c.DBDriver == "mysql" {
		c.MySQLDSN = buildMySQLDSN()
	}

	// Two named devices: IN (outside/entry) and OUT (inside/exit).
	for _, d := range []DeviceConfig{
		{
			Name:      getEnv("ZK_IN_NAME", "Outside (Entry)"),
			IP:        getEnv("ZK_IN_IP", ""),
			Port:      uint(getEnvInt("ZK_IN_PORT", 4370)),
			Direction: "in",
		},
		{
			Name:      getEnv("ZK_OUT_NAME", "Inside (Exit)"),
			IP:        getEnv("ZK_OUT_IP", ""),
			Port:      uint(getEnvInt("ZK_OUT_PORT", 4370)),
			Direction: "out",
		},
	} {
		if strings.TrimSpace(d.IP) != "" {
			c.Devices = append(c.Devices, d)
		}
	}

	if len(c.Devices) == 0 {
		return nil, fmt.Errorf("no devices configured: set ZK_IN_IP and/or ZK_OUT_IP")
	}

	return c, nil
}

func buildMySQLDSN() string {
	if dsn := os.Getenv("DB_DSN"); dsn != "" {
		return dsn
	}
	user := getEnv("DB_USERNAME", "root")
	pass := getEnv("DB_PASSWORD", "")
	host := getEnv("DB_HOST", "127.0.0.1")
	port := getEnv("DB_PORT", "3306")
	name := getEnv("DB_DATABASE", "attendance")
	// parseTime=true so timestamps map to time.Time; loc=Local keeps wall time.
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=true&loc=Local",
		user, pass, host, port, name)
}

func getEnv(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

func getEnvInt(key string, def int) int {
	if v, ok := os.LookupEnv(key); ok {
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
			return n
		}
	}
	return def
}

func getEnvBool(key string, def bool) bool {
	if v, ok := os.LookupEnv(key); ok {
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "1", "true", "yes", "on":
			return true
		case "0", "false", "no", "off":
			return false
		}
	}
	return def
}
