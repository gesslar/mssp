// Package mssp provides functionality for the MSSP protocol.
package mssp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"
)

// Telnet holds the Telnet and MSSP protocol constants as their actual byte
// values (0-255). Using byte types lets them be placed directly into a slice
// when building or matching the wire protocol.
var Telnet = struct {
	IAC, DO, SB, SE        byte
	MSSP, MSSPVar, MSSPVal byte
}{255, 253, 250, 240, 70, 1, 2}

// Byte sequences used to frame and request an MSSP subnegotiation on the wire:
//
//   - StartPattern marks the beginning of an MSSP payload (IAC SB MSSP).
//   - EndPattern marks the end of a subnegotiation (IAC SE).
//   - RequestSequence asks the server to begin MSSP (IAC DO MSSP).
var (
	StartPattern    = []byte{Telnet.IAC, Telnet.SB, Telnet.MSSP}
	EndPattern      = []byte{Telnet.IAC, Telnet.SE}
	RequestSequence = []byte{Telnet.IAC, Telnet.DO, Telnet.MSSP}
)

// ConnectionConfig describes how to reach a server: the host and port to dial
// and the connection timeout in seconds.
type ConnectionConfig struct {
	host    string
	port    int
	timeout int
}

// NewConnectionConfig returns a *ConnectionConfig populated with the given
// host, port, and timeout (in seconds).
func NewConnectionConfig(host string, port int, timeout int) *ConnectionConfig {
	return &ConnectionConfig{
		host:    host,
		port:    port,
		timeout: timeout,
	}
}

// Host returns the host the config will dial.
func (c *ConnectionConfig) Host() string { return c.host }

// Port returns the port the config will dial.
func (c *ConnectionConfig) Port() int { return c.port }

// Timeout returns the connection timeout in seconds.
func (c *ConnectionConfig) Timeout() int { return c.timeout }

// ParseMSSP decodes an MSSP subnegotiation payload (the bytes between
// IAC SB MSSP and IAC SE) into a map of variable name to its value(s).
//
// The wire format is a flat sequence of:
//
//	MSSP_VAR <name> MSSP_VAL <value> [ MSSP_VAR <name> MSSP_VAL <value> ... ]
//
// Per the MSSP spec there are two distinct mechanisms, which this function
// keeps separate:
//
//   - A single MSSP_VAR followed by several MSSP_VAL is a list. Each key maps
//     to a slice, so this becomes all the values: PORT -> ["80","23","3000"].
//   - The same MSSP_VAR repeated as separate tokens is an override; the last
//     one reported wins. NAME "my" then NAME "mud" yields NAME -> ["mud"].
//
// Each key maps to a slice; callers that want the effective (default) value
// should take the last element.
func ParseMSSP(payload []byte) Result {
	result := make(Result)

	var key string
	var cur []byte
	// state: true when accumulating a value, false when accumulating a name
	inValue := false
	haveKey := false

	commit := func() {
		if haveKey && inValue {
			result[key] = append(result[key], string(cur))
		}
	}

	for _, b := range payload {
		switch b {
		case Telnet.MSSPVar:
			// A new MSSP_VAR token begins a fresh group. Commit the value in
			// progress, then start reading a name.
			commit()
			key = ""
			cur = cur[:0]
			inValue = false
			haveKey = false
		case Telnet.MSSPVal:
			if inValue {
				// Repeated MSSP_VAL within one group: another value (a list).
				commit()
			} else {
				// First MSSP_VAL after a MSSP_VAR. A new VAR token overrides any
				// prior group for the same name (last reported wins), so drop
				// whatever was recorded before and start this group fresh.
				key = string(cur)
				haveKey = true
				delete(result, key)
				inValue = true
			}
			cur = cur[:0]
		default:
			cur = append(cur, b)
		}
	}
	commit()

	return result
}

// Result holds parsed MSSP data: each variable name mapped to its value(s).
// It stays lossless ([]string everywhere); the scalar-vs-array distinction is
// applied only when marshaling to JSON.
type Result map[string][]string

// MarshalJSON implements json.Marshaler so that json.Marshal(result) renders a
// key with a single value as a string and a key with multiple values as an
// array. Callers just use the standard library; this is invoked automatically.
func (r Result) MarshalJSON() ([]byte, error) {
	out := make(map[string]any, len(r))
	for k, v := range r {
		if len(v) == 1 {
			out[k] = v[0]
		} else {
			out[k] = v
		}
	}
	return json.Marshal(out)
}

// String implements fmt.Stringer for readable debug output, e.g. with
// fmt.Println or %v. A single value is shown bare and multiple values are
// bracketed: "NAME=Oxidus PORT=[80 23 3000]". Order follows map iteration and
// is not stable; callers that need a deterministic ordering should sort.
func (r Result) String() string {
	var b strings.Builder
	first := true
	for k, v := range r {
		if !first {
			b.WriteByte(' ')
		}
		first = false
		if len(v) == 1 {
			fmt.Fprintf(&b, "%s=%s", k, v[0])
		} else {
			fmt.Fprintf(&b, "%s=%v", k, v)
		}
	}
	return b.String()
}

// Connect dials the server described by config, requests MSSP, and reads until
// it finds a complete MSSP subnegotiation (framed by StartPattern and
// EndPattern). The enclosed payload is decoded with ParseMSSP and returned as a
// Result. It returns an error if the connection, the initial write, or a read
// before a complete payload arrives fails.
func Connect(config *ConnectionConfig) (Result, error) {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(config.host, fmt.Sprintf("%d", config.port)), time.Duration(config.timeout)*time.Second)
	if err != nil {
		return nil, err
	}

	defer conn.Close()

	_, err = conn.Write(RequestSequence)
	if err != nil {
		return nil, err
	}

	var buf []byte
	tmp := make([]byte, 512)
	var payload []byte

	var result Result

	if err := conn.SetReadDeadline(time.Now().Add(time.Duration(config.timeout) * time.Second)); err != nil {
		return nil, err
	}

	for {
		n, err := conn.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
		}

		_, after, ok := bytes.Cut(buf, StartPattern)
		if ok {
			before, _, ok := bytes.Cut(after, EndPattern)
			if ok {
				payload = before
				// got it, parse payload
				response := ParseMSSP(payload)
				result = response
				break
			}
		}

		if err != nil {
			return nil, err
		}
	}

	return result, nil
}
