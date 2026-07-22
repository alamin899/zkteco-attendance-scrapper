// Package zk is a pure-Go client for the ZKTeco device protocol. It ports the
// proven pyzk logic with strict packet framing so large attendance logs read
// reliably, and adds a reconnect-retry loop for transient failures.
package zk

import (
	"fmt"
	"strconv"
	"time"

	"attendance-collector/internal/config"
)

// Punch is one raw attendance record.
type Punch struct {
	UserPin string
	Time    time.Time
	State   int // verify mode (fingerprint/card/password)
	Type    int // punch type (0 check-in, 1 check-out, ...)
}

// User is one enrolled person on the device.
type User struct {
	UID       int
	UserPin   string
	Name      string
	Card      string
	Privilege int
}

// Client reads from a single device using shared options + reconnect retries.
type Client struct {
	dev config.DeviceConfig
	cfg *config.Config
	loc *time.Location
}

func New(dev config.DeviceConfig, cfg *config.Config) *Client {
	return &Client{dev: dev, cfg: cfg, loc: time.Local}
}

func (c *Client) password() int {
	n, _ := strconv.Atoi(c.cfg.Password)
	return n
}

// withConnection opens a fresh connection, runs fn, always disconnects, and
// retries the whole cycle on failure (reconnecting clears a desynced stream).
func (c *Client) withConnection(fn func(cn *conn) error) error {
	attempts := c.cfg.Retries + 1
	if attempts < 1 {
		attempts = 1
	}

	var lastErr error
	for attempt := 0; attempt < attempts; attempt++ {
		cn, err := dial(c.dev.Address(), c.cfg.Timeout, c.password())
		if err != nil {
			lastErr = err
		} else {
			err = fn(cn)
			cn.close()
			if err == nil {
				return nil
			}
			lastErr = err
		}
		if attempt < attempts-1 {
			time.Sleep(c.cfg.RetryDelay * time.Duration(attempt+1))
		}
	}
	return fmt.Errorf("%s: %w (after %d attempts)", c.dev.Address(), lastErr, attempts)
}

// Time returns the device clock (quick connectivity check).
func (c *Client) Time() (time.Time, error) {
	var out time.Time
	err := c.withConnection(func(cn *conn) error {
		code, payload, err := cn.command(cmdGetTime, nil)
		if err != nil {
			return err
		}
		if code != cmdAckOK || len(payload) < 12 {
			return fmt.Errorf("can't get time (code %d)", code)
		}
		out = decodeTime(payload[8:12], c.loc)
		return nil
	})
	return out, err
}

// Users returns everyone enrolled on the device.
func (c *Client) Users() ([]User, error) {
	var out []User
	err := c.withConnection(func(cn *conn) error {
		u, err := cn.users(c.loc)
		if err != nil {
			return err
		}
		out = u
		return nil
	})
	return out, err
}

// Attendance returns all raw punches currently stored on the device.
func (c *Client) Attendance() ([]Punch, error) {
	var out []Punch
	err := c.withConnection(func(cn *conn) error {
		p, err := cn.attendance(c.loc)
		if err != nil {
			return err
		}
		out = p
		return nil
	})
	return out, err
}

// ClearAttendance is permanently disabled to ensure device data is never cleared.
func (c *Client) ClearAttendance() error {
	return fmt.Errorf("ClearAttendance is disabled: hardware log deletion is prevented for data safety")
}

// --- high-level reads (mirror pyzk get_users / get_attendance) ---

// readSizes returns (users, records) counts from CMD_GET_FREE_SIZES.
func (cn *conn) readSizes() (users int, records int, err error) {
	code, payload, err := cn.command(cmdGetFreeSizes, nil)
	if err != nil {
		return 0, 0, err
	}
	if code != cmdAckOK {
		return 0, 0, fmt.Errorf("can't read sizes (code %d)", code)
	}
	data := payload[8:]
	if len(data) >= 80 {
		users = int(le.Uint32(data[16:20]))  // field[4]
		records = int(le.Uint32(data[32:36])) // field[8]
	}
	return users, records, nil
}

func (cn *conn) users(loc *time.Location) ([]User, error) {
	users, _, err := cn.readSizes()
	if err != nil {
		return nil, err
	}
	if users == 0 {
		return nil, nil
	}
	data, err := cn.readWithBuffer(cmdUserTempRRQ, fctUser, 0)
	if err != nil {
		return nil, err
	}
	if len(data) <= 4 {
		return nil, nil
	}
	total := int(le.Uint32(data[0:4]))
	data = data[4:]
	size := 72
	if users > 0 && total/users == 28 {
		size = 28
	}

	var out []User
	if size == 28 {
		for len(data) >= 28 {
			r := data[:28]
			out = append(out, User{
				UID:       int(le.Uint16(r[0:2])),
				Privilege: int(r[2]),
				Name:      cstr(r[8:16]),
				Card:      cardString(le.Uint32(r[16:20])),
				UserPin:   strconv.Itoa(int(le.Uint32(r[24:28]))),
			})
			data = data[28:]
		}
	} else {
		for len(data) >= 72 {
			r := data[:72]
			out = append(out, User{
				UID:       int(le.Uint16(r[0:2])),
				Privilege: int(r[2]),
				Name:      cstr(r[11:35]),
				Card:      cardString(le.Uint32(r[35:39])),
				UserPin:   cstr(r[48:72]),
			})
			data = data[72:]
		}
	}
	return out, nil
}

func (cn *conn) attendance(loc *time.Location) ([]Punch, error) {
	_, records, err := cn.readSizes()
	if err != nil {
		return nil, err
	}
	if records == 0 {
		return nil, nil
	}
	data, err := cn.readWithBuffer(cmdAttlogRRQ, 0, 0)
	if err != nil {
		return nil, err
	}
	if len(data) < 4 {
		return nil, nil
	}
	total := int(le.Uint32(data[0:4]))
	data = data[4:]

	recordSize := 40
	if records > 0 && total/records > 0 {
		recordSize = total / records
	}

	var out []Punch
	switch recordSize {
	case 8:
		for len(data) >= 8 {
			r := data[:8]
			out = append(out, Punch{
				UserPin: strconv.Itoa(int(le.Uint16(r[0:2]))),
				State:   int(r[2]),
				Time:    decodeTime(r[3:7], loc),
				Type:    int(r[7]),
			})
			data = data[8:]
		}
	case 16:
		for len(data) >= 16 {
			r := data[:16]
			out = append(out, Punch{
				UserPin: strconv.Itoa(int(le.Uint32(r[0:4]))),
				Time:    decodeTime(r[4:8], loc),
				State:   int(r[8]),
				Type:    int(r[9]),
			})
			data = data[16:]
		}
	default: // 40
		for len(data) >= 40 {
			r := data[:40]
			out = append(out, Punch{
				UserPin: cstr(r[2:26]),
				State:   int(r[26]),
				Time:    decodeTime(r[27:31], loc),
				Type:    int(r[31]),
			})
			data = data[40:]
		}
	}
	return out, nil
}

func cardString(card uint32) string {
	if card == 0 {
		return ""
	}
	return strconv.FormatUint(uint64(card), 10)
}
