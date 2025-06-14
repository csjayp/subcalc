package subcalc

import (
	"fmt"
	"math"
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
	curr  net.IP
	count float64
	index float64
}

func NewIPRangeStreamer(start net.IP, bits int) *IPRangeStreamer {
	b := IPWIDTH - bits
	count := 1 << b
	ipCopy := make(net.IP, len(start))
	copy(ipCopy, start)
	return &IPRangeStreamer{
		curr:  ipCopy,
		count: float64(count),
		index: 0,
	}
}

func NewIP6RangeStreamer(start net.IP, bits int) *IPRangeStreamer {
	b := IPV6WIDTH - bits
	count := math.Pow(2, float64(b))
	ipCopy := make(net.IP, len(start))
	copy(ipCopy, start)
	return &IPRangeStreamer{
		curr:  ipCopy,
		count: count,
		index: 0,
	}
}

func ChunkToPart(chunk []string, chunkID int, lastBlock bool) string {
	respChunk := strings.Join(chunk, ",")
	if len(chunk) < 32 && chunkID != 0 ||
		len(chunk) == 32 && !lastBlock {
		respChunk = respChunk + ","
	}
	return respChunk
}

func (it *IPRangeStreamer) Next() (string, bool) {
	if it.index >= it.count {
		return "", false
	}
	ipStr := it.curr.String()
	IncrementIP(it.curr)
	it.index++
	return ipStr, true
}

func (it *IPRangeStreamer) Finished() bool {
	return it.index == it.count
}

func (it *IPRangeStreamer) NextBatch() ([]string, bool) {
	if it.index >= it.count {
		return nil, false
	}
	batch := make([]string, 0, 32)
	for i := 0; i < 32 && it.index < it.count; i++ {
		ent := "\"" + it.curr.String() + "\""
		batch = append(batch, ent)
		IncrementIP(it.curr)
		it.index++
	}
	return batch, true
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
	mask := make([]byte, 16)
	for i := 0; i < bits/8; i++ {
		mask[i] = 0xFF
	}
	if bits%8 != 0 {
		mask[bits/8] = 0xFF << (8 - bits%8)
	}
	if af == AF_INET {
		return mask[:4]
	}
	return mask[:16]
}

func ApplyMask(ip net.IP, mask net.IPMask) net.IP {
	res := make(net.IP, len(ip))
	for i := 0; i < len(mask); i++ {
		res[i] = ip[i] & mask[i]
	}
	return res
}

func SetMaskBits(ip net.IP, b int) net.IP {
	res := make(net.IP, len(ip))
	copy(res, ip)
	for i := len(ip) - 1; b > 0 && i >= 0; i-- {
		if b >= 8 {
			res[i] |= 0xFF
			b -= 8
		} else {
			res[i] |= (1<<b - 1)
			break
		}
	}
	return res
}

func RangeIPv6(start net.IP, mask net.IPMask, target net.IP) []string {
	// NB: we need this to be a stream instead of a complete buffer
	ret := make([]string, 0)
	curr := make(net.IP, len(start))
	copy(curr, start)
	for {
		if !MatchMasked(curr, mask, target) {
			break
		}
		ipstr := fmt.Sprintf("%v", curr)
		ret = append(ret, ipstr)
		IncrementIP(curr)
	}
	return ret
}

func RangeIPv4(start net.IP, b int) []string {
	count := 1 << b
	curr := make(net.IP, len(start))
	ret := make([]string, 0)
	copy(curr, start)
	for i := 0; i < count; i++ {
		ipstr := fmt.Sprintf("%v", curr)
		IncrementIP(curr)
		ret = append(ret, ipstr)
	}
	return ret
}

func MatchMasked(a net.IP, mask net.IPMask, ref net.IP) bool {
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
