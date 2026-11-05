package subcalc

import (
	"bytes"
	"errors"
	"net"
	"testing"
)

func TestMakeMaskBounds(t *testing.T) {
	for _, bits := range []int{-1, 33} {
		if MakeMask(AF_INET, bits) != nil {
			t.Fatalf("MakeMask inet %d should reject", bits)
		}
	}
	if got := MakeMask(AF_INET, 0); len(got) != 4 {
		t.Fatalf("/0 mask length = %d", len(got))
	}
	if got := MakeMask(AF_INET, 32); len(got) != 4 {
		t.Fatalf("/32 mask length = %d", len(got))
	}
}

type failWriter struct{}

func (failWriter) Write([]byte) (int, error) { return 0, errors.New("stop") }

func TestStreamingRangeAndWriterFailure(t *testing.T) {
	it := NewIPRangeStreamer(net.ParseIP("192.0.2.0"), 30)
	first, ok := it.Next()
	if !ok || first != "192.0.2.0" {
		t.Fatalf("first = %q, %v", first, ok)
	}
	batch, ok := it.NextBatch()
	if !ok || len(batch) != 3 {
		t.Fatalf("batch = %v, %v", batch, ok)
	}
	if _, ok := it.Next(); ok || !it.Finished() {
		t.Fatal("stream did not exhaust")
	}
	var out bytes.Buffer
	if err := WriteRangeIPv4(&out, net.ParseIP("192.0.2.0"), 30); err != nil || out.String() != "192.0.2.0\n192.0.2.1\n192.0.2.2\n192.0.2.3\n" {
		t.Fatalf("write range: %q, %v", out.String(), err)
	}
	if err := WriteRangeIPv6(failWriter{}, net.ParseIP("2001:db8::"), 128); err == nil {
		t.Fatal("expected writer failure")
	}
}

func TestStreamerNetworkBoundaries(t *testing.T) {
	cases := []struct {
		name   string
		start  string
		prefix int
		ipv6   bool
		want   []string
	}{
		{"v4 /31", "192.0.2.3", 31, false, []string{"192.0.2.2", "192.0.2.3"}},
		{"v4 /32", "192.0.2.3", 32, false, []string{"192.0.2.3"}},
		{"v6 /127", "2001:db8::3", 127, true, []string{"2001:db8::2", "2001:db8::3"}},
		{"v6 /128", "2001:db8::3", 128, true, []string{"2001:db8::3"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var it *IPRangeStreamer
			if tc.ipv6 {
				it = NewIP6RangeStreamer(net.ParseIP(tc.start), tc.prefix)
			} else {
				it = NewIPRangeStreamer(net.ParseIP(tc.start), tc.prefix)
			}
			for _, want := range tc.want {
				got, ok := it.Next()
				if !ok || got != want {
					t.Fatalf("Next()=%q,%v want %q", got, ok, want)
				}
			}
			if _, ok := it.Next(); ok || !it.Finished() {
				t.Fatal("did not exhaust after final address")
			}
		})
	}
}

func TestInvalidStreamers(t *testing.T) {
	for _, it := range []*IPRangeStreamer{NewIPRangeStreamer(net.ParseIP("::1"), 24), NewIPRangeStreamer(net.ParseIP("1.2.3.4"), -1), NewIPRangeStreamer(net.ParseIP("1.2.3.4"), 33), NewIP6RangeStreamer(net.ParseIP("1.2.3.4"), 64), NewIP6RangeStreamer(net.ParseIP("2001:db8::1"), -1), NewIP6RangeStreamer(net.ParseIP("2001:db8::1"), 129)} {
		if _, ok := it.Next(); ok {
			t.Fatal("invalid streamer yielded an address")
		}
	}
}

func TestIPv4IPv6ZeroPrefixStreamsLazily(t *testing.T) {
	for _, tc := range []struct {
		name  string
		start net.IP
		make  func(net.IP, int) *IPRangeStreamer
		first string
	}{
		{"ipv4", net.ParseIP("192.0.2.7"), NewIPRangeStreamer, "0.0.0.0"},
		{"ipv6", net.ParseIP("2001:db8::7"), NewIP6RangeStreamer, "::"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			it := tc.make(tc.start, 0)
			got, ok := it.Next()
			if !ok || got != tc.first {
				t.Fatalf("first=%q,%v want %q", got, ok, tc.first)
			}
			if it.Finished() {
				t.Fatal("/0 stream eagerly exhausted")
			}
			next := "0.0.0.1"
			if tc.name == "ipv6" {
				next = "::1"
			}
			if !it.curr.Equal(net.ParseIP(next)) {
				t.Fatalf("unexpected next address state: %v", it.curr)
			}
		})
	}
}

func TestRangeIPv6StopsAfterAddressSpaceWrap(t *testing.T) {
	got := RangeIPv6(net.ParseIP("ffff:ffff:ffff:ffff:ffff:ffff:ffff:fffe"), net.CIDRMask(0, 128), net.IPv6zero)
	if len(got) != 2 || got[1] != "ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff" {
		t.Fatalf("wrapped range=%v", got)
	}
}

func TestLegacyRangeHelpers(t *testing.T) {
	if got := RangeIPv4(net.ParseIP("192.0.2.0"), 2); len(got) != 4 || got[3] != "192.0.2.3" {
		t.Fatalf("IPv4 legacy range: %v", got)
	}
	got := RangeIPv6(net.ParseIP("2001:db8::"), net.CIDRMask(127, 128), net.ParseIP("2001:db8::1"))
	if len(got) != 2 || got[0] != "2001:db8::" || got[1] != "2001:db8::1" {
		t.Fatalf("IPv6 legacy range: %v", got)
	}
}

func TestMakeMaskChecked(t *testing.T) {
	for _, tc := range []struct {
		af      AddressFamily
		bits    int
		wantLen int
		wantErr bool
	}{
		{AF_INET, 0, 4, false}, {AF_INET, 32, 4, false}, {AF_INET, -1, 0, true}, {AF_INET, 33, 0, true},
		{AF_INET6, 0, 16, false}, {AF_INET6, 128, 16, false}, {AF_INET6, -1, 0, true}, {AF_INET6, 129, 0, true},
		{AddressFamily(99), 0, 0, true},
	} {
		got, err := MakeMaskChecked(tc.af, tc.bits)
		if (err != nil) != tc.wantErr || len(got) != tc.wantLen {
			t.Errorf("MakeMaskChecked(%v,%d) = len %d, err %v", tc.af, tc.bits, len(got), err)
		}
	}
}

func TestRangeIPv4BoundsAndOrder(t *testing.T) {
	got := RangeIPv4(net.ParseIP("192.0.2.0"), 2)
	if len(got) != 4 || got[0] != "192.0.2.0" || got[3] != "192.0.2.3" {
		t.Fatalf("range = %v", got)
	}
	if got := RangeIPv4(net.ParseIP("192.0.2.0"), 33); got != nil {
		t.Fatalf("invalid range = %v", got)
	}
}

func TestRangeIPv6RejectsBadShapes(t *testing.T) {
	if got := RangeIPv6(net.IP{1}, net.IPMask{255}, net.IP{1}); got != nil {
		t.Fatalf("range = %v", got)
	}
}

func TestAddressHelpersIPv4AndIPv6(t *testing.T) {
	tests := []struct {
		ip    net.IP
		width int
	}{
		{net.ParseIP("192.0.2.1"), 32},
		{net.ParseIP("2001:db8::1"), 128},
	}
	for _, tc := range tests {
		if got := SetMaskBits(tc.ip, 0); !got.Equal(tc.ip) {
			t.Errorf("zero bits: %v", got)
		}
		if got := SetMaskBits(tc.ip, tc.width); got == nil {
			t.Errorf("all bits returned nil")
		} else {
			for _, b := range got {
				if b != 0xff {
					t.Errorf("all bits = %v", got)
					break
				}
			}
		}
		if got := SetMaskBits(tc.ip, 9); got == nil {
			t.Errorf("partial boundary returned nil")
		}
		mask := MakeMask(AF_INET6, tc.width)
		if tc.width == IPWIDTH {
			mask = MakeMask(AF_INET, tc.width)
		}
		if got := ApplyMask(tc.ip, mask); got == nil {
			t.Errorf("ApplyMask failed for %v", tc.ip)
		}
		if !MatchMasked(tc.ip, mask, tc.ip) {
			t.Errorf("MatchMasked failed for %v", tc.ip)
		}
	}
	if SetMaskBits(net.IP{1}, 1) != nil || ApplyMask(net.IP{1}, net.IPMask{255}) != nil || MatchMasked(net.IP{1}, net.IPMask{255}, net.IP{1}) {
		t.Fatal("invalid/mismatched inputs were accepted")
	}
}
