package zk

import (
	"encoding/binary"
	"fmt"
	"net"
	"time"
)

// ZKTeco proprietary protocol constants (from the pyzk reference implementation).
const (
	cmdConnect      = 1000
	cmdExit         = 1001
	cmdAuth         = 1102
	cmdAckOK        = 2000
	cmdAckUnauth    = 2005
	cmdPrepareData  = 1500
	cmdData         = 1501
	cmdFreeData     = 1502
	cmdDataWRRQ     = 1503
	cmdReadBuffer   = 1504
	cmdUserTempRRQ  = 9
	cmdAttlogRRQ    = 13
	cmdClearAttlog  = 15
	cmdGetFreeSizes = 50
	cmdGetTime      = 201

	fctUser = 5

	machinePrep1 = 0x5050
	machinePrep2 = 0x7d82
	ushrtMax     = 65535
	maxChunk     = 0xFFC0
)

var le = binary.LittleEndian

// conn is a low-level, framed connection to a device. Every reply is read as a
// complete TCP packet (8-byte "top" giving the length, then exactly that many
// bytes), which is what keeps the stream in sync on high-volume devices.
type conn struct {
	sock      net.Conn
	sessionID uint16
	replyID   uint16
	timeout   time.Duration
}

// dial opens a connection and performs the CMD_CONNECT (+ optional auth) handshake.
func dial(addr string, timeout time.Duration, password int) (*conn, error) {
	s, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return nil, err
	}
	c := &conn{sock: s, replyID: ushrtMax - 1, timeout: timeout}

	code, payload, err := c.command(cmdConnect, nil)
	if err != nil {
		c.sock.Close()
		return nil, err
	}
	c.sessionID = le.Uint16(payload[4:6])

	if code == cmdAckUnauth {
		key := makeCommKey(password, int(c.sessionID), 50)
		code, _, err = c.command(cmdAuth, key)
		if err != nil {
			c.sock.Close()
			return nil, err
		}
	}
	if code != cmdAckOK {
		c.sock.Close()
		return nil, fmt.Errorf("connect refused (code %d)", code)
	}
	return c, nil
}

func (c *conn) close() {
	if c.sock != nil {
		_, _, _ = c.command(cmdExit, nil)
		_ = c.sock.Close()
		c.sock = nil
	}
}

// buildHeader assembles command header + payload, computing the checksum and
// advancing the reply id, exactly like pyzk's __create_header.
func (c *conn) buildHeader(command uint16, data []byte) []byte {
	buf := make([]byte, 8+len(data))
	le.PutUint16(buf[0:], command)
	le.PutUint16(buf[2:], 0)
	le.PutUint16(buf[4:], c.sessionID)
	le.PutUint16(buf[6:], c.replyID)
	copy(buf[8:], data)

	cs := checksum(buf)

	c.replyID++
	if c.replyID >= ushrtMax {
		c.replyID -= ushrtMax
	}

	le.PutUint16(buf[2:], cs)
	le.PutUint16(buf[6:], c.replyID)
	return buf
}

// command sends one command and returns the reply code + full payload
// (payload[0:8] is the reply header, payload[8:] is the data).
func (c *conn) command(command uint16, data []byte) (uint16, []byte, error) {
	hdr := c.buildHeader(command, data)
	top := make([]byte, 8+len(hdr))
	le.PutUint16(top[0:], machinePrep1)
	le.PutUint16(top[2:], machinePrep2)
	le.PutUint32(top[4:], uint32(len(hdr)))
	copy(top[8:], hdr)

	_ = c.sock.SetWriteDeadline(time.Now().Add(c.timeout))
	if _, err := c.sock.Write(top); err != nil {
		return 0, nil, err
	}
	return c.readPacket()
}

// readPacket reads exactly one full protocol packet.
func (c *conn) readPacket() (uint16, []byte, error) {
	top, err := c.readN(8)
	if err != nil {
		return 0, nil, err
	}
	if le.Uint16(top[0:]) != machinePrep1 || le.Uint16(top[2:]) != machinePrep2 {
		return 0, nil, fmt.Errorf("TCP packet invalid")
	}
	size := int(le.Uint32(top[4:]))
	if size < 8 {
		return 0, nil, fmt.Errorf("short packet (%d bytes)", size)
	}
	payload, err := c.readN(size)
	if err != nil {
		return 0, nil, err
	}
	code := le.Uint16(payload[0:])
	c.replyID = le.Uint16(payload[6:])
	return code, payload, nil
}

// readN reads exactly n bytes, refreshing the deadline on each read so a slow
// but steady transfer of a large log doesn't trip the timeout.
func (c *conn) readN(n int) ([]byte, error) {
	buf := make([]byte, n)
	got := 0
	for got < n {
		_ = c.sock.SetReadDeadline(time.Now().Add(c.timeout))
		m, err := c.sock.Read(buf[got:])
		got += m
		if err != nil {
			if got >= n {
				break
			}
			return nil, err
		}
	}
	return buf, nil
}

// readWithBuffer implements the buffered read (CMD_DATA_WRRQ) used for large
// payloads (users, attendance).
func (c *conn) readWithBuffer(command uint16, fct, ext int) ([]byte, error) {
	cs := make([]byte, 11)
	cs[0] = 1
	le.PutUint16(cs[1:], command)
	le.PutUint32(cs[3:], uint32(fct))
	le.PutUint32(cs[7:], uint32(ext))

	code, payload, err := c.command(cmdDataWRRQ, cs)
	if err != nil {
		return nil, err
	}
	data := payload[8:]

	if code == cmdData {
		return data, nil
	}
	if code != cmdAckOK && code != cmdPrepareData {
		return nil, fmt.Errorf("buffered read not supported (code %d)", code)
	}
	if len(data) < 5 {
		return nil, fmt.Errorf("bad prepare response")
	}

	size := int(le.Uint32(data[1:5]))
	out := make([]byte, 0, size)
	start, remaining := 0, size
	for remaining > 0 {
		chunk := remaining
		if chunk > maxChunk {
			chunk = maxChunk
		}
		d, err := c.readChunk(start, chunk)
		if err != nil {
			return nil, err
		}
		out = append(out, d...)
		start += chunk
		remaining -= chunk
	}
	_, _, _ = c.command(cmdFreeData, nil)
	return out, nil
}

func (c *conn) readChunk(start, size int) ([]byte, error) {
	cs := make([]byte, 8)
	le.PutUint32(cs[0:], uint32(start))
	le.PutUint32(cs[4:], uint32(size))

	var lastErr error
	for retry := 0; retry < 3; retry++ {
		code, payload, err := c.command(cmdReadBuffer, cs)
		if err != nil {
			lastErr = err
			continue
		}
		data, err := c.receiveChunk(code, payload)
		if err != nil {
			lastErr = err
			continue
		}
		return data, nil
	}
	return nil, fmt.Errorf("can't read chunk %d:%d: %v", start, size, lastErr)
}

func (c *conn) receiveChunk(code uint16, payload []byte) ([]byte, error) {
	data := payload[8:]
	switch code {
	case cmdData:
		return data, nil
	case cmdPrepareData:
		if len(data) < 4 {
			return nil, fmt.Errorf("bad chunk prepare")
		}
		size := int(le.Uint32(data[0:4]))
		out := make([]byte, 0, size)
		for len(out) < size {
			c2, p2, err := c.readPacket()
			if err != nil {
				return nil, err
			}
			if c2 != cmdData {
				break
			}
			out = append(out, p2[8:]...)
		}
		// Trailing CMD_ACK_OK.
		_, _, _ = c.readPacket()
		return out, nil
	default:
		return nil, fmt.Errorf("invalid chunk response %d", code)
	}
}

// checksum mirrors pyzk's __create_checksum (16-bit one's-complement sum).
func checksum(p []byte) uint16 {
	var sum int
	i, l := 0, len(p)
	for l > 1 {
		sum += int(le.Uint16(p[i : i+2]))
		i += 2
		l -= 2
		if sum > ushrtMax {
			sum -= ushrtMax
		}
	}
	if l > 0 {
		sum += int(p[i])
	}
	for sum > ushrtMax {
		sum -= ushrtMax
	}
	sum = ^sum
	for sum < 0 {
		sum += ushrtMax
	}
	return uint16(sum)
}

// makeCommKey scrambles the device password + session id (commpro.c MakeKey).
func makeCommKey(key, sessionID, ticks int) []byte {
	k := 0
	for i := 0; i < 32; i++ {
		if key&(1<<i) != 0 {
			k = (k << 1) | 1
		} else {
			k = k << 1
		}
	}
	k += sessionID

	b := make([]byte, 4)
	le.PutUint32(b, uint32(k))
	b[0] ^= 'Z'
	b[1] ^= 'K'
	b[2] ^= 'S'
	b[3] ^= 'O'
	// swap the two 16-bit halves
	b = []byte{b[2], b[3], b[0], b[1]}
	B := byte(ticks & 0xff)
	return []byte{b[0] ^ B, b[1] ^ B, B, b[3] ^ B}
}

// decodeTime converts the packed 4-byte device timestamp to a time.Time in loc.
func decodeTime(b []byte, loc *time.Location) time.Time {
	t := int(le.Uint32(b))
	second := t % 60
	t /= 60
	minute := t % 60
	t /= 60
	hour := t % 24
	t /= 24
	day := t%31 + 1
	t /= 31
	month := t%12 + 1
	t /= 12
	year := t + 2000
	return time.Date(year, time.Month(month), day, hour, minute, second, 0, loc)
}

func cstr(b []byte) string {
	for i, ch := range b {
		if ch == 0 {
			return string(b[:i])
		}
	}
	return string(b)
}
