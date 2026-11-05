package subcalc

import (
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
)

const (
	IPWIDTH   = 32
	IPV6WIDTH = 128
)

type AddressFamily int

const (
	AF_INET AddressFamily = iota
	AF_INET6
)

func (w AddressFamily) String() string {
	switch w {
	case AF_INET:
		return "inet"
	case AF_INET6:
		return "inet6"
	default:
		return "invalid"
	}
}

type IPRangeStreamer struct {
	curr     net.IP
	last     net.IP
	finished bool
}

func NewIPRangeStreamer(start net.IP, bits int) *IPRangeStreamer {
	if start.To4() == nil || bits < 0 || bits > IPWIDTH {
		return &IPRangeStreamer{}
	}
	ip := start.To4()
	ipCopy := make(net.IP, len(ip))
	copy(ipCopy, ip)
	mask := net.CIDRMask(bits, IPWIDTH)
	first := ApplyMask(ipCopy, mask)
	return &IPRangeStreamer{curr: first, last: SetMaskBits(first, IPWIDTH-bits)}
}

func NewIP6RangeStreamer(start net.IP, bits int) *IPRangeStreamer {
	if start.To16() == nil || start.To4() != nil || bits < 0 || bits > IPV6WIDTH {
		return &IPRangeStreamer{}
	}
	ip := start.To16()
	ipCopy := make(net.IP, len(ip))
	copy(ipCopy, ip)
	mask := net.CIDRMask(bits, IPV6WIDTH)
	first := ApplyMask(ipCopy, mask)
	return &IPRangeStreamer{curr: first, last: SetMaskBits(first, IPV6WIDTH-bits)}
}

func ChunkToPart(chunk []string, lastBlock bool) string {
	respChunk := strings.Join(chunk, ",")
	if !lastBlock {
		respChunk = respChunk + ","
	}
	return respChunk
}

func (it *IPRangeStreamer) Next() (string, bool) {
	if it == nil || len(it.curr) == 0 {
		return "", false
	}
	if it.finished {
		return "", false
	}
	ipStr := it.curr.String()
	it.advance()
	return ipStr, true
}

func (it *IPRangeStreamer) advance() {
	if it.curr.Equal(it.last) {
		it.finished = true
		return
	}
	IncrementIP(it.curr)
}

func (it *IPRangeStreamer) Finished() bool {
	if it == nil || len(it.curr) == 0 {
		return true
	}
	return it.finished
}

func (it *IPRangeStreamer) NextBatch() ([]string, bool) {
	if it == nil || len(it.curr) == 0 {
		return nil, false
	}
	if it.finished {
		return nil, false
	}
	batch := make([]string, 0, 32)
	for i := 0; i < 32 && !it.finished; i++ {
		ent := "\"" + it.curr.String() + "\""
		batch = append(batch, ent)
		it.advance()
	}
	return batch, true
}

func WriteRangeIPv4(w io.Writer, start net.IP, prefixBits int) error {
	if prefixBits < 0 || prefixBits > IPWIDTH || start.To4() == nil {
		return fmt.Errorf("invalid IPv4 range")
	}
	return writeRange(w, NewIPRangeStreamer(start, prefixBits))
}
func WriteRangeIPv6(w io.Writer, start net.IP, prefixBits int) error {
	if prefixBits < 0 || prefixBits > IPV6WIDTH || start.To16() == nil || start.To4() != nil {
		return fmt.Errorf("invalid IPv6 range")
	}
	return writeRange(w, NewIP6RangeStreamer(start, prefixBits))
}
func writeRange(w io.Writer, it *IPRangeStreamer) error {
	for {
		ip, ok := it.Next()
		if !ok {
			return nil
		}
		_, err := fmt.Fprintln(w, ip)
		if err != nil {
			return err
		}
	}
}

func InvertMask(ip net.IP) (net.IPMask, string) {
	if ip4 := ip.To4(); ip4 != nil {
		inv := make(net.IPMask, 4)
		for i := 0; i < 4; i++ {
			inv[i] = ^ip4[i]
		}
		return inv, net.IP(inv).String()
	}

	ip6 := ip.To16()
	if ip6 == nil {
		panic("invertMask: invalid IP address")
	}
	inv := make(net.IPMask, 16)
	for i := 0; i < 16; i++ {
		inv[i] = ^ip6[i]
	}
	return inv, net.IP(inv).String()
}

func MakeMask(af AddressFamily, bits int) net.IPMask {
	mask, _ := MakeMaskChecked(af, bits)
	return mask
}

func MakeMaskChecked(af AddressFamily, bits int) (net.IPMask, error) {
	max := 0
	size := 0
	switch af {
	case AF_INET:
		max, size = IPWIDTH, 4
	case AF_INET6:
		max, size = IPV6WIDTH, 16
	default:
		return nil, errors.New("unsupported address family")
	}
	if bits < 0 || bits > max {
		return nil, fmt.Errorf("prefix length %d out of range for %s (0-%d)", bits, af, max)
	}
	mask := make([]byte, 16)
	for i := 0; i < bits/8; i++ {
		mask[i] = 0xFF
	}
	if bits%8 != 0 {
		mask[bits/8] = 0xFF << (8 - bits%8)
	}
	return mask[:size], nil
}

func ApplyMask(ip net.IP, mask net.IPMask) net.IP {
	if len(mask) == 4 {
		ip = ip.To4()
	} else if len(mask) == 16 {
		if ip.To4() != nil {
			return nil
		}
		ip = ip.To16()
	} else {
		return nil
	}
	if ip == nil {
		return nil
	}
	res := make(net.IP, len(mask))
	for i := 0; i < len(mask); i++ {
		res[i] = ip[i] & mask[i]
	}
	return res
}

func SetMaskBits(ip net.IP, b int) net.IP {
	width := 0
	if ip.To4() != nil {
		ip = ip.To4()
		width = IPWIDTH
	} else if ip.To16() != nil {
		ip = ip.To16()
		width = IPV6WIDTH
	} else {
		return nil
	}
	if b < 0 || b > width {
		return nil
	}
	res := append(net.IP(nil), ip...)
	for i := len(res) - 1; b > 0; i-- {
		n := b
		if n > 8 {
			n = 8
		}
		res[i] |= byte((1 << uint(n)) - 1)
		b -= n
	}
	return res
}

func RangeIPv6(start net.IP, mask net.IPMask, target net.IP) []string {
	if len(start) != net.IPv6len || len(target) != net.IPv6len || len(mask) != net.IPv6len {
		return nil
	}
	ret := make([]string, 0)
	curr := make(net.IP, len(start))
	copy(curr, start)
	for MatchMasked(curr, mask, target) {
		ipstr := fmt.Sprintf("%v", curr)
		ret = append(ret, ipstr)
		IncrementIP(curr)
		if curr.Equal(net.IPv6zero) {
			break // include the final address once, then stop at wraparound
		}
	}
	return ret
}

// RangeIPv4 materializes the whole range and can exhaust memory for large inputs;
// use WriteRangeIPv4 for streamed output.
func RangeIPv4(start net.IP, b int) []string {
	if start.To4() == nil || b < 0 || b > IPWIDTH {
		return nil
	}
	start = start.To4()
	count := uint64(1) << uint(b)
	curr := make(net.IP, len(start))
	ret := make([]string, 0)
	copy(curr, start)
	for i := uint64(0); i < count; i++ {
		ipstr := fmt.Sprintf("%v", curr)
		IncrementIP(curr)
		ret = append(ret, ipstr)
	}
	return ret
}

func MatchMasked(a net.IP, mask net.IPMask, ref net.IP) bool {
	if len(mask) == 4 {
		a, ref = a.To4(), ref.To4()
	} else if len(mask) == 16 {
		if a.To4() != nil || ref.To4() != nil {
			return false
		}
		a, ref = a.To16(), ref.To16()
	} else {
		return false
	}
	if a == nil || ref == nil {
		return false
	}
	for i := 0; i < len(mask); i++ {
		if (a[i] & mask[i]) != (ref[i] & mask[i]) {
			return false
		}
	}
	return true
}

func IncrementIP(ip net.IP) {
	for i := len(ip) - 1; i >= 0; i-- {
		ip[i]++
		if ip[i] != 0 {
			break
		}
	}
}
