package mssp

import (
	"encoding/json"
	"net"
	"reflect"
	"testing"
	"time"
)

// mssp builds an MSSP subnegotiation payload (the bytes that live between
// StartPattern and EndPattern) from a flat token list. Each pair is emitted as
// MSSP_VAR name MSSP_VAL value; to express a multi-value list or an override,
// pass the appropriate token sequence via the helpers below.
func varTok(name string) []byte { return append([]byte{Telnet.MSSPVar}, []byte(name)...) }
func valTok(val string) []byte  { return append([]byte{Telnet.MSSPVal}, []byte(val)...) }

func concat(chunks ...[]byte) []byte {
	var out []byte
	for _, c := range chunks {
		out = append(out, c...)
	}
	return out
}

func TestParseMSSP(t *testing.T) {
	tests := []struct {
		name    string
		payload []byte
		want    Result
	}{
		{
			name:    "empty payload",
			payload: nil,
			want:    Result{},
		},
		{
			name:    "single var single val",
			payload: concat(varTok("NAME"), valTok("Oxidus")),
			want:    Result{"NAME": {"Oxidus"}},
		},
		{
			name: "multiple distinct vars",
			payload: concat(
				varTok("NAME"), valTok("Oxidus"),
				varTok("PLAYERS"), valTok("42"),
			),
			want: Result{"NAME": {"Oxidus"}, "PLAYERS": {"42"}},
		},
		{
			name: "single var multiple vals is a list",
			payload: concat(
				varTok("PORT"), valTok("80"), valTok("23"), valTok("3000"),
			),
			want: Result{"PORT": {"80", "23", "3000"}},
		},
		{
			name: "repeated var override keeps last",
			payload: concat(
				varTok("NAME"), valTok("my"),
				varTok("NAME"), valTok("mud"),
			),
			want: Result{"NAME": {"mud"}},
		},
		{
			name: "repeated var override drops earlier list",
			payload: concat(
				varTok("PORT"), valTok("80"), valTok("23"),
				varTok("PORT"), valTok("3000"),
			),
			want: Result{"PORT": {"3000"}},
		},
		{
			name:    "var with empty value",
			payload: concat(varTok("NAME"), valTok("")),
			want:    Result{"NAME": {""}},
		},
		{
			name:    "var with no value is dropped",
			payload: varTok("NAME"),
			want:    Result{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseMSSP(tt.payload)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseMSSP() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestResultMarshalJSON(t *testing.T) {
	tests := []struct {
		name string
		r    Result
		want string
	}{
		{
			name: "single value renders as scalar string",
			r:    Result{"NAME": {"Oxidus"}},
			want: `{"NAME":"Oxidus"}`,
		},
		{
			name: "multiple values render as array",
			r:    Result{"PORT": {"80", "23"}},
			want: `{"PORT":["80","23"]}`,
		},
		{
			name: "empty result renders as empty object",
			r:    Result{},
			want: `{}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.r)
			if err != nil {
				t.Fatalf("json.Marshal() error = %v", err)
			}
			if string(got) != tt.want {
				t.Errorf("json.Marshal() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestResultString(t *testing.T) {
	// Single-key cases are deterministic; map ordering only matters with >1 key.
	if got := (Result{"NAME": {"Oxidus"}}).String(); got != "NAME=Oxidus" {
		t.Errorf("String() = %q, want %q", got, "NAME=Oxidus")
	}
	if got := (Result{"PORT": {"80", "23", "3000"}}).String(); got != "PORT=[80 23 3000]" {
		t.Errorf("String() = %q, want %q", got, "PORT=[80 23 3000]")
	}
	if got := (Result{}).String(); got != "" {
		t.Errorf("String() = %q, want empty", got)
	}
}

func TestNewConnectionConfig(t *testing.T) {
	got := NewConnectionConfig("example.com", 4000, 7)
	want := &ConnectionConfig{host: "example.com", port: 4000, timeout: 7}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("NewConnectionConfig() = %#v, want %#v", got, want)
	}
}

// startMSSPServer launches a one-shot TCP server that, after the client sends
// any data, replies with frame wrapped in StartPattern/EndPattern (plus the
// optional surrounding noise) and closes. It returns the dialable host:port.
func startMSSPServer(t *testing.T, prefix, payload, suffix []byte) (string, int) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { ln.Close() })

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		buf := make([]byte, 64)
		_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		_, _ = conn.Read(buf) // wait for the client's RequestSequence
		frame := concat(prefix, StartPattern, payload, EndPattern, suffix)
		_, _ = conn.Write(frame)
	}()

	addr := ln.Addr().(*net.TCPAddr)
	return addr.IP.String(), addr.Port
}

func TestConnect(t *testing.T) {
	payload := concat(varTok("NAME"), valTok("Oxidus"), varTok("PLAYERS"), valTok("3"))
	// Surround the MSSP frame with unrelated bytes to ensure Connect locates it.
	host, port := startMSSPServer(t, []byte("welcome\r\n"), payload, []byte("trailing"))

	cfg := NewConnectionConfig(host, port, 2)
	got, err := Connect(cfg)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	want := Result{"NAME": {"Oxidus"}, "PLAYERS": {"3"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Connect() = %#v, want %#v", got, want)
	}
}

func TestConnectDialError(t *testing.T) {
	// Reserve a port with a listener, then close it so the dial is refused.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()

	cfg := NewConnectionConfig("127.0.0.1", port, 1)
	if _, err := Connect(cfg); err == nil {
		t.Error("Connect() expected error for refused connection, got nil")
	}
}
