package main

import (
	"encoding/binary"
	"fmt"
	"math"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/csjayp/subcalc/subcalc-go/pkg/subcalc"
)

var dorange int

type cmdargs struct {
	af   subcalc.AddressFamily
	addr string
	bits uint
}

func usage() {
	fmt.Fprintln(os.Stderr, "Usage: subcalc [inet|inet6] address[/prefixlen] [print]")
	os.Exit(1)
}

func main() {
	if len(os.Args) == 1 {
		usage()
	}

	var cd cmdargs
	proccmdargs(os.Args, &cd)

	if cd.af == subcalc.AF_INET6 {
		adr6 := net.ParseIP(cd.addr).To16()
		if adr6 == nil {
			fmt.Fprintln(os.Stderr, "Invalid IPv6 address")
			os.Exit(1)
		}

		ip6 := make(net.IP, len(adr6))
		copy(ip6, adr6)
		ip6mask := subcalc.MakeMask(subcalc.AF_INET6, int(cd.bits))
		b := subcalc.IPV6WIDTH - int(cd.bits)

		rangeStart := subcalc.ApplyMask(ip6, ip6mask)
		rangeEnd := subcalc.SetMaskBits(rangeStart, b)

		fmt.Printf("%srange:       %s > %s\n", semicolon(), rangeStart, rangeEnd)
		hosts := math.Pow(2, float64(b))
		fmt.Printf("%shosts:       %.0f\n", semicolon(), hosts)
		fmt.Printf("%sprefixlen:   %d\n", semicolon(), cd.bits)
		fmt.Printf("%smask:        %s\n", semicolon(), ip6mask.String())

		if dorange == 0 {
			return
		}

		printRangeIPv6(rangeStart, ip6mask, ip6)
	} else if cd.af == subcalc.AF_INET {
		ip := net.ParseIP(cd.addr).To4()
		if ip == nil {
			fmt.Fprintln(os.Stderr, "Invalid IPv4 address")
			os.Exit(1)
		}

		ip1 := make(net.IP, len(ip))
		ip2 := make(net.IP, len(ip))
		copy(ip1, ip)
		copy(ip2, ip)

		b := subcalc.IPWIDTH - int(cd.bits)

		maskBytes := subcalc.MakeMask(subcalc.AF_INET, int(cd.bits))
		rangeStart := subcalc.ApplyMask(ip1, net.IPv4Mask(maskBytes[0], maskBytes[1], maskBytes[2], maskBytes[3]))

		rangeEnd := subcalc.SetMaskBits(rangeStart, b)

		r1 := binary.BigEndian.Uint32(rangeStart.To4())
		r2 := binary.BigEndian.Uint32(rangeEnd.To4())
		fmt.Printf("%srange:       %s > %s\n", semicolon(), rangeStart, rangeEnd)
		fmt.Printf("%srange b10:   %d > %d\n", semicolon(), r1, r2)
		fmt.Printf("%srange b16:   0x%x > 0x%x\n", semicolon(), r1, r2)

		p := math.Pow(2, float64(b))
		fmt.Printf("%shosts:       %.0f\n", semicolon(), p)
		fmt.Printf("%sprefixlen:   %d\n", semicolon(), cd.bits)

		maskBytes = subcalc.MakeMask(subcalc.AF_INET, int(cd.bits))
		netmask := net.IPv4Mask(maskBytes[0], maskBytes[1], maskBytes[2], maskBytes[3])

		fmt.Printf("%snetmask:     %s\n", semicolon(), net.IP(netmask).String())
		_, ms := subcalc.InvertMask(net.IP(netmask))
		fmt.Printf("%smask:        %s\n", semicolon(), ms)

		if dorange == 0 {
			return
		}
		printRangeIPv4(rangeStart, b)
	}
}

func proccmdargs(args []string, cd *cmdargs) {
	var af subcalc.AddressFamily

	if len(args) < 3 {
		usage()
	}
	if args[len(args)-1] == "print" {
		dorange = 1
	}
	afStr := args[1]
	spec := args[2]
	if afStr == "inet" {
		af = subcalc.AF_INET
	} else if afStr == "inet6" {
		af = subcalc.AF_INET6
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

func printRangeIPv6(start net.IP, mask net.IPMask, target net.IP) {
	// NB: stream instead of buffer
	list := subcalc.RangeIPv6(start, mask, target)
	for _, ip := range list {
		fmt.Printf("%s\n", ip)
	}
}

func printRangeIPv4(start net.IP, b int) {
	list := subcalc.RangeIPv4(start, b)
	for _, ip := range list {
		fmt.Printf("%s\n", ip)
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
