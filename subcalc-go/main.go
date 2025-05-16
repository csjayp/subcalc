package main

import (
	"encoding/binary"
	"fmt"
	"math"
	"net"
	"os"
	"strconv"
	"strings"
)

const (
	IPWIDTH   = 32
	IPV6WIDTH = 128
)

var dorange int

type cmdargs struct {
	af   int
	addr string
	bits uint
}

const (
	AF_INET = iota
	AF_INET6
)

func usage() {
	fmt.Fprintln(os.Stderr, "Usage: subcalc [inet|inet6] address[/prefixlen] [print]")
	os.Exit(1)
}

func invertMask(ip net.IP) (net.IPMask, string) {
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

func main() {
	if len(os.Args) == 1 {
		usage()
	}

	var cd cmdargs
	proccmdargs(os.Args, &cd)

	if cd.af == AF_INET6 {
		adr6 := net.ParseIP(cd.addr).To16()
		if adr6 == nil {
			fmt.Fprintln(os.Stderr, "Invalid IPv6 address")
			os.Exit(1)
		}

		ip6 := make(net.IP, len(adr6))
		copy(ip6, adr6)
		ip6mask := makeMask(AF_INET6, int(cd.bits))
		b := IPV6WIDTH - int(cd.bits)

		rangeStart := applyMask(ip6, ip6mask)
		rangeEnd := setMaskBits(rangeStart, b)

		fmt.Printf("%srange:       %s > %s\n", semicolon(), rangeStart, rangeEnd)
		hosts := math.Pow(2, float64(b))
		fmt.Printf("%shosts:       %.0f\n", semicolon(), hosts)
		fmt.Printf("%sprefixlen:   %d\n", semicolon(), cd.bits)
		fmt.Printf("%smask:        %s\n", semicolon(), ip6mask.String())

		if dorange == 0 {
			return
		}

		printRangeIPv6(rangeStart, ip6mask, ip6)
	} else if cd.af == AF_INET {
		ip := net.ParseIP(cd.addr).To4()
		if ip == nil {
			fmt.Fprintln(os.Stderr, "Invalid IPv4 address")
			os.Exit(1)
		}

		ip1 := make(net.IP, len(ip))
		ip2 := make(net.IP, len(ip))
		copy(ip1, ip)
		copy(ip2, ip)

		b := IPWIDTH - int(cd.bits)

		maskBytes := makeMask(AF_INET, int(cd.bits))
		rangeStart := applyMask(ip1, net.IPv4Mask(maskBytes[0], maskBytes[1], maskBytes[2], maskBytes[3]))

		rangeEnd := setMaskBits(rangeStart, b)

		r1 := binary.BigEndian.Uint32(rangeStart.To4())
		r2 := binary.BigEndian.Uint32(rangeEnd.To4())
		fmt.Printf("%srange:       %s > %s\n", semicolon(), rangeStart, rangeEnd)
		fmt.Printf("%srange b10:   %d > %d\n", semicolon(), r1, r2)
		fmt.Printf("%srange b16:   0x%x > 0x%x\n", semicolon(), r1, r2)

		p := math.Pow(2, float64(b))
		fmt.Printf("%shosts:       %.0f\n", semicolon(), p)
		fmt.Printf("%sprefixlen:   %d\n", semicolon(), cd.bits)

		maskBytes = makeMask(AF_INET, int(cd.bits))
		netmask := net.IPv4Mask(maskBytes[0], maskBytes[1], maskBytes[2], maskBytes[3])

		fmt.Printf("%snetmask:     %s\n", semicolon(), net.IP(netmask).String())
		_, ms := invertMask(net.IP(netmask))
		fmt.Printf("%smask:        %s\n", semicolon(), ms)

		if dorange == 0 {
			return
		}
		printRangeIPv4(rangeStart, b)
	}
}

func proccmdargs(args []string, cd *cmdargs) {
	if len(args) < 3 {
		usage()
	}
	if args[len(args)-1] == "print" {
		dorange = 1
	}
	afStr := args[1]
	spec := args[2]
	var af int
	if afStr == "inet" {
		af = AF_INET
	} else if afStr == "inet6" {
		af = AF_INET6
	} else {
		errExit("Invalid address family")
	}
	addr, bits := parseCIDR(spec)
	cd.af = af
	cd.addr = addr
	cd.bits = uint(bits)
}

func parseCIDR(s string) (string, int) {
	parts := strings.Split(s, "/")
	if len(parts) != 2 {
		errExit("CIDR format required: address/prefix")
	}
	bits, err := strconv.Atoi(parts[1])
	if err != nil {
		errExit("Invalid prefix length")
	}
	return parts[0], bits
}

func makeMask(af int, bits int) net.IPMask {
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

func applyMask(ip net.IP, mask net.IPMask) net.IP {
	res := make(net.IP, len(ip))
	for i := 0; i < len(mask); i++ {
		res[i] = ip[i] & mask[i]
	}
	return res
}

func setMaskBits(ip net.IP, b int) net.IP {
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

func rangeIPv6(start net.IP, mask net.IPMask, target net.IP) []string {
	// NB: we need this to be a stream instead of a complete buffer
	ret := make([]string, 0)
	curr := make(net.IP, len(start))
	copy(curr, start)
	for {
		if !matchMasked(curr, mask, target) {
			break
		}
		ipstr := fmt.Sprintf("%v", curr)
		ret = append(ret, ipstr)
		incrementIP(curr)
	}
	return ret
}

func printRangeIPv6(start net.IP, mask net.IPMask, target net.IP) {
	// NB: stream instead of buffer
	list := rangeIPv6(start, mask, target)
	for _, ip := range list {
		fmt.Printf("%s\n", ip)
	}
}

func rangeIPv4(start net.IP, b int) []string {
	count := 1 << b
	curr := make(net.IP, len(start))
	ret := make([]string, 0)
	copy(curr, start)
	for i := 0; i < count; i++ {
		ipstr := fmt.Sprintf("%v", curr)
		incrementIP(curr)
		ret = append(ret, ipstr)
	}
	return ret
}

func printRangeIPv4(start net.IP, b int) {
	list := rangeIPv4(start, b)
	for _, ip := range list {
		fmt.Printf("%s\n", ip)
	}
}

func matchMasked(a net.IP, mask net.IPMask, ref net.IP) bool {
	for i := 0; i < len(mask); i++ {
		if (a[i] & mask[i]) != (ref[i] & mask[i]) {
			return false
		}
	}
	return true
}

func incrementIP(ip net.IP) {
	for i := len(ip) - 1; i >= 0; i-- {
		ip[i]++
		if ip[i] != 0 {
			break
		}
	}
}

func semicolon() string {
	if dorange > 0 {
		return "; "
	}
	return ""
}

func errExit(msg string) {
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(1)
}
