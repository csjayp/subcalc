package main

import (
	"bytes"
	"errors"
	"io"
	"net"
	"strings"
	"testing"

	"github.com/csjayp/subcalc/subcalc-go/pkg/subcalc"
)

func TestParseArgs(t *testing.T) {
	cases := []struct {
		name    string
		args    []string
		af      subcalc.AddressFamily
		wantErr string
	}{
		{"ipv4", []string{"subcalc", "inet", "192.0.2.1/24"}, subcalc.AF_INET, ""},
		{"ipv6 print", []string{"subcalc", "inet6", "2001:db8::1/64", "print"}, subcalc.AF_INET6, ""},
		{"count", []string{"subcalc"}, 0, "Usage:"},
		{"family", []string{"subcalc", "bad", "192.0.2.1/24"}, 0, "address family"},
		{"cidr", []string{"subcalc", "inet", "192.0.2.1"}, 0, "CIDR"},
		{"prefix syntax", []string{"subcalc", "inet", "192.0.2.1/no"}, 0, "prefix length"},
		{"prefix range", []string{"subcalc", "inet", "192.0.2.1/33"}, 0, "0-32"},
		{"optional", []string{"subcalc", "inet", "192.0.2.1/24", "oops"}, 0, "optional argument"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cd, err := parseArgs(tc.args)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("err=%v", err)
				}
				return
			}
			if err != nil || cd.af != tc.af {
				t.Fatalf("args=%+v err=%v", cd, err)
			}
		})
	}
}

type errorWriter struct{}

func (errorWriter) Write([]byte) (int, error) { return 0, errors.New("write failed") }

type shortOutputWriter struct{}

func (shortOutputWriter) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	return len(p) - 1, nil
}

func TestExecuteReturnsSummaryAndRangeWriteErrors(t *testing.T) {
	for _, tc := range []cmdargs{
		{af: subcalc.AF_INET, addr: "192.0.2.1", bits: 32},
		{af: subcalc.AF_INET, addr: "192.0.2.1", bits: 32, print: true},
	} {
		if err := execute(tc, errorWriter{}); err == nil {
			t.Fatalf("execute(%+v) did not return writer error", tc)
		}
	}
	if err := subcalc.WriteRangeIPv4(errorWriter{}, []byte{192, 0, 2, 1}, 0); err == nil {
		t.Fatal("range writer error lost")
	}
}

func TestExecuteDetectsShortWrites(t *testing.T) {
	if err := execute(cmdargs{af: subcalc.AF_INET, addr: "192.0.2.1", bits: 32}, shortOutputWriter{}); err != io.ErrShortWrite {
		t.Fatalf("summary error=%v", err)
	}
	if err := subcalc.WriteRangeIPv4(shortOutputWriter{}, net.ParseIP("192.0.2.1"), 0); err != io.ErrShortWrite {
		t.Fatalf("stream error=%v", err)
	}
}

func TestExecuteStableIPv4Output(t *testing.T) {
	var out bytes.Buffer
	if err := execute(cmdargs{af: subcalc.AF_INET, addr: "192.0.2.7", bits: 24}, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(out.String(), "range:       192.0.2.0 > 192.0.2.255\n") {
		t.Fatalf("output changed: %q", out.String())
	}
}
