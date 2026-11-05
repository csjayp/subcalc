package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/csjayp/subcalc/subcalc-go/pkg/subcalc"
)

type cmdargs struct {
	af    subcalc.AddressFamily
	addr  string
	bits  uint
	print bool
}

func writef(out io.Writer, format string, args ...any) error {
	line := []byte(fmt.Sprintf(format, args...))
	n, err := out.Write(line)
	if err != nil {
		return err
	}
	if n < len(line) {
		return io.ErrShortWrite
	}
	return nil
}

func main() {
	if err := run(os.Args, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, out io.Writer) error {
	cd, err := parseArgs(args)
	if err != nil {
		return err
	}
	return execute(cd, out)
}

func parseArgs(args []string) (cmdargs, error) {
	if len(args) < 3 || len(args) > 4 {
		return cmdargs{}, fmt.Errorf("Usage: subcalc [inet|inet6] address[/prefixlen] [print]")
	}
	var af subcalc.AddressFamily
	switch args[1] {
	case "inet":
		af = subcalc.AF_INET
	case "inet6":
		af = subcalc.AF_INET6
	default:
		return cmdargs{}, fmt.Errorf("Invalid address family")
	}
	addr, bits, err := parseCIDR(args[2])
	if err != nil {
		return cmdargs{}, err
	}
	max := subcalc.IPWIDTH
	if af == subcalc.AF_INET6 {
		max = subcalc.IPV6WIDTH
	}
	if bits < 0 || bits > max {
		return cmdargs{}, fmt.Errorf("Invalid prefix length for %s (0-%d)", af, max)
	}
	printRange := false
	if len(args) == 4 {
		if args[3] != "print" {
			return cmdargs{}, fmt.Errorf("Invalid optional argument: %s", args[3])
		}
		printRange = true
	}
	return cmdargs{af: af, addr: addr, bits: uint(bits), print: printRange}, nil
}

func execute(cd cmdargs, out io.Writer) error {
	ip, err := parseAddress(cd.af, cd.addr)
	if err != nil {
		return err
	}
	sep := ""
	if cd.print {
		sep = "; "
	}
	switch cd.af {
	case subcalc.AF_INET6:
		mask, err := subcalc.MakeMaskChecked(cd.af, int(cd.bits))
		if err != nil {
			return err
		}
		b := subcalc.IPV6WIDTH - int(cd.bits)
		start := subcalc.ApplyMask(ip, mask)
		end := subcalc.SetMaskBits(start, b)
		if err := writef(out, "%srange:       %s > %s\n", sep, start, end); err != nil {
			return err
		}
		if err := writef(out, "%shosts:       %.0f\n", sep, math.Pow(2, float64(b))); err != nil {
			return err
		}
		if err := writef(out, "%sprefixlen:   %d\n", sep, cd.bits); err != nil {
			return err
		}
		if err := writef(out, "%smask:        %s\n", sep, mask.String()); err != nil {
			return err
		}
		if cd.print {
			return subcalc.WriteRangeIPv6(out, start, int(cd.bits))
		}
	case subcalc.AF_INET:
		mask, err := subcalc.MakeMaskChecked(cd.af, int(cd.bits))
		if err != nil {
			return err
		}
		b := subcalc.IPWIDTH - int(cd.bits)
		start := subcalc.ApplyMask(ip, mask)
		end := subcalc.SetMaskBits(start, b)
		r1, r2 := binary.BigEndian.Uint32(start.To4()), binary.BigEndian.Uint32(end.To4())
		if err := writef(out, "%srange:       %s > %s\n", sep, start, end); err != nil {
			return err
		}
		if err := writef(out, "%srange b10:   %d > %d\n", sep, r1, r2); err != nil {
			return err
		}
		if err := writef(out, "%srange b16:   0x%x > 0x%x\n", sep, r1, r2); err != nil {
			return err
		}
		if err := writef(out, "%shosts:       %.0f\n", sep, math.Pow(2, float64(b))); err != nil {
			return err
		}
		if err := writef(out, "%sprefixlen:   %d\n", sep, cd.bits); err != nil {
			return err
		}
		netmask := net.IPv4Mask(mask[0], mask[1], mask[2], mask[3])
		if err := writef(out, "%snetmask:     %s\n", sep, net.IP(netmask).String()); err != nil {
			return err
		}
		_, ms := subcalc.InvertMask(net.IP(netmask))
		if err := writef(out, "%smask:        %s\n", sep, ms); err != nil {
			return err
		}
		if cd.print {
			return subcalc.WriteRangeIPv4(out, start, b)
		}
	}
	return nil
}

func parseCIDR(s string) (string, int, error) {
	parts := strings.Split(s, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", 0, fmt.Errorf("CIDR format required: address/prefix")
	}
	bits, err := strconv.Atoi(parts[1])
	if err != nil {
		return "", 0, fmt.Errorf("Invalid prefix length")
	}
	return parts[0], bits, nil
}

func parseAddress(af subcalc.AddressFamily, addr string) (net.IP, error) {
	parsed := net.ParseIP(addr)
	if parsed == nil {
		return nil, fmt.Errorf("Invalid IP address")
	}
	switch af {
	case subcalc.AF_INET:
		ip := parsed.To4()
		if ip == nil {
			return nil, fmt.Errorf("Invalid IPv4 address")
		}
		return ip, nil
	case subcalc.AF_INET6:
		ip := parsed.To16()
		if ip == nil || parsed.To4() != nil {
			return nil, fmt.Errorf("Invalid IPv6 address")
		}
		return ip, nil
	default:
		return nil, fmt.Errorf("Invalid address family")
	}
}
