package subcalc

import (
	"fmt"
	"net"
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

func InvertMask(ip net.IP) (net.IPMask, string) {
	ip4 := ip.To4()
	if ip4 == nil {
		panic("invertMask: only IPv4 addresses supported")
	}
	inv := make(net.IPMask, 4)
	for i := 0; i < 4; i++ {
		inv[i] = ^ip4[i]
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
