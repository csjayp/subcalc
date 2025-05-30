package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/csjayp/subcalc/subcalc-go/pkg/subcalc"
	"github.com/fastly/compute-sdk-go/fsthttp"
)

type subcalcRange struct {
	First string `json:"first_address"`
	Last  string `json:"last_address"`
}

type subcalcBlock struct {
	AddressRange    subcalcRange `json:"address_range"`
	AddressRangeB10 subcalcRange `json:"address_range_base10"`
	AddressRangeB16 subcalcRange `json:"address_range_base16"`
	HostCount       string       `json:"host_count"`
	PrefixLength    int          `json:"prefix_length"`
	NetMask         string       `json:"network_mask"`
	Mask            string       `json:"mask"`
}

type subcalcInput struct {
	Family  subcalc.AddressFamily `json:"address_family"`
	Address string                `json:"address"`
	Bits    int                   `json:"cidr_bits"`
}

type subcalcResponse struct {
	SubcalcQuery  subcalcInput `json:"subcalc_query"`
	SubcalcAnswer subcalcBlock `json:"subcalc_answer"`
}

func uriToSubcalcInput(uri string) (subcalcInput, error) {
	var err error

	comps := strings.Split(uri, "/")
	if len(comps) != 4 {
		return subcalcInput{}, errors.New("invalid uri " + uri)
	}
	ret := subcalcInput{}
	switch strings.ToLower(comps[1]) {
	case "inet":
		ret.Family = subcalc.AF_INET
	case "inet6":
		ret.Family = subcalc.AF_INET6
	default:
		return ret, errors.New("invalid address family")
	}
	ret.Address, err = url.QueryUnescape(comps[2])
	if err != nil {
		return ret, err
	}
	ret.Bits, err = strconv.Atoi(comps[3])
	if err != nil {
		return ret, err
	}
	if ret.Bits > 128 || ret.Bits < 1 {
		return ret, errors.New("invalid CIDR specification")
	}
	return ret, nil
}

func handleInet6(input subcalcInput) (subcalcBlock, error) {
	adr6 := net.ParseIP(input.Address).To16()
	if adr6 == nil {
		return subcalcBlock{}, errors.New("invalid ip6 address")
	}
	ip6 := make(net.IP, len(adr6))
	copy(ip6, adr6)
	ip6mask := subcalc.MakeMask(subcalc.AF_INET6, int(input.Bits))
	b := subcalc.IPV6WIDTH - int(input.Bits)
	thePow := big.NewInt(int64(b))
	hosts := new(big.Int).Exp(big.NewInt(2), thePow, nil)
	rangeStart := subcalc.ApplyMask(ip6, ip6mask)
	rangeEnd := subcalc.SetMaskBits(rangeStart, b)
	_, mask := subcalc.InvertMask(net.IP(ip6mask))
	subcalcResp := subcalcBlock{
		AddressRange: subcalcRange{
			First: rangeStart.String(),
			Last:  rangeEnd.String(),
		},
		PrefixLength: input.Bits,
		HostCount:    hosts.String(),
		Mask:         mask,
		NetMask:      ip6mask.String(),
	}
	return subcalcResp, nil
}

func handleInet4(input subcalcInput) (subcalcBlock, error) {
	ip := net.ParseIP(input.Address).To4()
	if ip == nil {
		return subcalcBlock{}, errors.New("invalid ip4 address")
	}
	ip1 := make(net.IP, len(ip))
	ip2 := make(net.IP, len(ip))
	copy(ip1, ip)
	copy(ip2, ip)
	b := subcalc.IPWIDTH - int(input.Bits)
	maskBytes := subcalc.MakeMask(subcalc.AF_INET, int(input.Bits))
	rangeStart := subcalc.ApplyMask(
		ip1,
		net.IPv4Mask(maskBytes[0], maskBytes[1], maskBytes[2], maskBytes[3]))
	rangeEnd := subcalc.SetMaskBits(rangeStart, b)
	r1 := binary.BigEndian.Uint32(rangeStart.To4())
	r2 := binary.BigEndian.Uint32(rangeEnd.To4())
	maskBytes = subcalc.MakeMask(subcalc.AF_INET, int(input.Bits))
	netmask := net.IPv4Mask(maskBytes[0], maskBytes[1], maskBytes[2], maskBytes[3])
	netmask_as_ip := net.IP(netmask).String()
	_, mask := subcalc.InvertMask(net.IP(netmask))
	hostCountStr := fmt.Sprintf("%.0f", math.Pow(2, float64(b)))
	subcalcResp := subcalcBlock{
		NetMask:      netmask_as_ip,
		Mask:         mask,
		PrefixLength: input.Bits,
		HostCount:    hostCountStr,
		AddressRange: subcalcRange{
			First: rangeStart.String(),
			Last:  rangeEnd.String(),
		},
		AddressRangeB10: subcalcRange{
			First: fmt.Sprintf("%d", r1),
			Last:  fmt.Sprintf("%d", r2),
		},
		AddressRangeB16: subcalcRange{
			First: fmt.Sprintf("0x%x", r1),
			Last:  fmt.Sprintf("0x%x", r2),
		},
	}
	return subcalcResp, nil
}

func main() {
	fsthttp.ServeFunc(func(ctx context.Context, w fsthttp.ResponseWriter, r *fsthttp.Request) {
		if r.Method == "POST" || r.Method == "PUT" || r.Method == "PATCH" || r.Method == "DELETE" {
			w.WriteHeader(fsthttp.StatusMethodNotAllowed)
			fmt.Fprintf(w, "This method is not allowed\n")
			return
		}
		input, err := uriToSubcalcInput(r.URL.Path)
		if err != nil {
			w.WriteHeader(fsthttp.StatusNotFound)
			fmt.Fprintf(w, "%s\r\n", err)
			return
		}
		resp := subcalcResponse{}
		switch input.Family {
		case subcalc.AF_INET6:
			block, err := handleInet6(input)
			if err != nil {
				w.WriteHeader(fsthttp.StatusBadRequest)
				fmt.Fprintf(w, "%s\r\n", err)
				return
			}
			resp = subcalcResponse{SubcalcAnswer: block, SubcalcQuery: input}
		case subcalc.AF_INET:
			block, err := handleInet4(input)
			if err != nil {
				w.WriteHeader(fsthttp.StatusBadRequest)
				fmt.Fprintf(w, "%s\r\n", err)
				return
			}
			resp = subcalcResponse{SubcalcAnswer: block, SubcalcQuery: input}
		}
		respBody, err := json.Marshal(resp)
		if err != nil {
			w.WriteHeader(fsthttp.StatusNotFound)
			fmt.Fprintf(w, "%s\r\n", err)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		io.Copy(w, io.NopCloser(bytes.NewReader(respBody)))
		return
	})
}
