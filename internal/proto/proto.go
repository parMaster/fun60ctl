// Package proto builds and parses the RY5088 vendor feature reports that stock
// firmware accepts in normal (application) mode. It does no I/O.
package proto

import (
	"errors"
	"fmt"
)

// ReportLen is the size of every vendor feature report, without the report ID.
const ReportLen = 64

// Opcodes used by fun60ctl. A read opcode is its write opcode with the high bit set.
const (
	OpGetInfo    byte = 0x8F
	OpSetProfile byte = 0x04
	OpGetProfile byte = 0x84
	OpFetchReply byte = 0xFC
)

// LayerCount is how many actuation layers the board stores (0..3 on the wire).
const LayerCount = 4

// Frame builds a 64-byte report with the Bit7 header checksum:
// bytes 0..7 sum to 0xFF modulo 256, and the checksum sits in byte 7.
func Frame(payload ...byte) [ReportLen]byte {
	var r [ReportLen]byte
	copy(r[:7], payload)
	var sum byte
	for _, b := range r[:7] {
		sum += b
	}
	r[7] = 0xFF - sum
	return r
}

// GetInfo asks for the dev_id and firmware version.
func GetInfo() [ReportLen]byte { return Frame(OpGetInfo) }

// GetLayer asks for the active actuation layer.
func GetLayer() [ReportLen]byte { return Frame(OpGetProfile) }

// SetLayer switches the actuation layer. layer is 0-based (wire value).
func SetLayer(layer byte) ([ReportLen]byte, error) {
	if layer >= LayerCount {
		return [ReportLen]byte{}, fmt.Errorf("layer %d out of range 0..%d", layer, LayerCount-1)
	}
	return Frame(OpSetProfile, layer), nil
}

// FetchReply asks the 2.4 GHz dongle for the keyboard's reply to the last
// command; the next GET_REPORT returns it. Unlike other frames it carries no
// checksum: the vendor app sends FC followed by zeros.
func FetchReply() [ReportLen]byte {
	var r [ReportLen]byte
	r[0] = OpFetchReply
	return r
}

// Info is the decoded GET_INFOR reply.
type Info struct {
	DevID   uint16
	Version string
}

// ErrBadReply means the reply did not echo the expected opcode.
var ErrBadReply = errors.New("unexpected reply")

// echoed returns the reply body starting at the opcode echo. Some platforms
// keep the leading report ID 0 in the buffer, so it tries both shapes.
func echoed(resp []byte, op byte, minLen int) ([]byte, error) {
	if len(resp) >= minLen && resp[0] == op {
		return resp, nil
	}
	if len(resp) >= minLen+1 && resp[0] == 0 && resp[1] == op {
		return resp[1:], nil
	}
	return nil, fmt.Errorf("%w: want echo 0x%02X, got % X", ErrBadReply, op, head(resp))
}

// Echoes reports whether resp starts with op, with or without a leading report ID 0.
func Echoes(resp []byte, op byte) bool {
	_, err := echoed(resp, op, 1)
	return err == nil
}

func head(b []byte) []byte {
	if len(b) > 9 {
		return b[:9]
	}
	return b
}

// ParseInfo decodes a GET_INFOR reply: dev_id is u16 LE in bytes 1..2,
// the version is major in byte 8 and minor in byte 7 (printed as "v306").
func ParseInfo(resp []byte) (Info, error) {
	r, err := echoed(resp, OpGetInfo, 9)
	if err != nil {
		return Info{}, err
	}
	return Info{
		DevID:   uint16(r[1]) | uint16(r[2])<<8,
		Version: fmt.Sprintf("v%d%02d", r[8], r[7]),
	}, nil
}

// ParseLayer decodes a GET_PROFILE reply into the 0-based actuation layer.
func ParseLayer(resp []byte) (byte, error) {
	r, err := echoed(resp, OpGetProfile, 2)
	if err != nil {
		return 0, err
	}
	if r[1] >= LayerCount {
		return 0, fmt.Errorf("%w: layer %d out of range", ErrBadReply, r[1])
	}
	return r[1], nil
}
