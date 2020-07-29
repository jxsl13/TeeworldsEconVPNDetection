package main

import (
	"encoding/binary"
	"errors"
	"net"
	"regexp"
)

var (
	ipv4SubnetWithReasonRegex = regexp.MustCompile(`([\d]{1,3}\.[\d]{1,3}\.[\d]{1,3}\.[\d]{1,3}\/[\d]{1,2})([\s]*#[\s]*(.*))`)
	ipv4SubnetRegex           = regexp.MustCompile(`[\d]{1,3}\.[\d]{1,3}\.[\d]{1,3}\.[\d]{1,3}\/[\d]{1,2}`)
	ipv4WithReasonRegex       = regexp.MustCompile(`([\d]{1,3}\.[\d]{1,3}\.[\d]{1,3}\.[\d]{1,3})([\s]*#[\s]*(.*))`)
	ipv4Regex                 = regexp.MustCompile(`[\d]{1,3}\.[\d]{1,3}\.[\d]{1,3}\.[\d]{1,3}`)

	formatDistinctionRegex          = regexp.MustCompile(`([\d]{1,3}\.[\d]{1,3}\.[\d]{1,3}\.[\d]{1,3})[^\d]*([\d]{1,3}\.[\d]{1,3}\.[\d]{1,3}\.[\d]{1,3})`)
	ipv4CustomRangesWithReasonRegex = regexp.MustCompile(`([\d]{1,3}\.[\d]{1,3}\.[\d]{1,3}\.[\d]{1,3})[\s]*-[\s]*([\d]{1,3}\.[\d]{1,3}\.[\d]{1,3}\.[\d]{1,3})([\s]*#[\s]*(.*))`)
	ipv4CustomRangesRegex           = regexp.MustCompile(`([\d]{1,3}\.[\d]{1,3}\.[\d]{1,3}\.[\d]{1,3})[\s]*-[\s]*([\d]{1,3}\.[\d]{1,3}\.[\d]{1,3}\.[\d]{1,3})`)
)

// if the IP has no custom reason, the default reason is used.
// in this case the default reason should be either "1" or "0"
func parseIPLine(line, defaultReason string) (ipsWithReasons map[string]string) {
	// 0.0.0.0 -> len = 7
	if len(line) < 7 {
		return nil
	}

	match := formatDistinctionRegex.FindStringSubmatch(line)
	if len(match) == 3 {

		match = ipv4CustomRangesWithReasonRegex.FindStringSubmatch(line)
		if len(match) == 5 {
			lower := match[1]
			upper := match[2]
			reason := match[4]

			ips, _ := ipsFromRange(lower, upper)
			ipsWithReasons = make(map[string]string, len(ips))
			for _, ip := range ips {
				ipsWithReasons[ip] = reason
			}
			return
		}

		match = ipv4CustomRangesRegex.FindStringSubmatch(line)
		if len(match) == 3 {
			lower := match[1]
			upper := match[2]

			ips, _ := ipsFromRange(lower, upper)
			ipsWithReasons = make(map[string]string, len(ips))
			for _, ip := range ips {
				ipsWithReasons[ip] = defaultReason
			}
			return
		}
		return nil
	}

	match = ipv4SubnetWithReasonRegex.FindStringSubmatch(line)
	if len(match) == 4 {
		ipSubnet := match[1]
		reason := match[3]
		ipList, err := ipsFromCIDR(ipSubnet)
		if err != nil {
			return nil
		}
		ipsWithReasons = make(map[string]string, len(ipList))

		for _, ip := range ipList {
			ipsWithReasons[ip] = reason
		}
		return
	}

	match = ipv4SubnetRegex.FindStringSubmatch(line)
	if len(match) == 1 {
		ipSubnet := match[0]
		reason := defaultReason
		ipList, err := ipsFromCIDR(ipSubnet)
		if err != nil {
			return nil
		}
		ipsWithReasons = make(map[string]string, len(ipList))

		for _, ip := range ipList {
			ipsWithReasons[ip] = reason
		}
		return
	}

	match = ipv4WithReasonRegex.FindStringSubmatch(line)
	if len(match) == 4 {
		ip := match[1]
		reason := match[3]
		ipsWithReasons = make(map[string]string, 1)
		ipsWithReasons[ip] = reason
		return
	}

	match = ipv4Regex.FindStringSubmatch(line)
	if len(match) == 1 {
		ip := match[0]
		reason := defaultReason
		ipsWithReasons = make(map[string]string, 1)
		ipsWithReasons[ip] = reason
		return
	}
	return nil
}

func ipsFromCIDR(cidr string) ([]string, error) {
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, err
	}

	var ips []string
	for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); inc(ip) {
		ips = append(ips, ip.String())
	}
	// remove network address and broadcast address
	return ips[1 : len(ips)-1], nil
}

func ipsFromRange(lowerBound, upperBound string) ([]string, error) {
	lower := net.ParseIP(lowerBound)
	upper := net.ParseIP(upperBound)
	if lower == nil || upper == nil {
		return nil, errors.New("either lower or upper bound are not valid IPs")
	}

	result := make([]string, 0, 4)

	inc(lower)
	for i := lower; !i.Equal(upper); inc(i) {

		// remove network address and broadcast address

		result = append(result, i.String())

	}
	return result, nil
}

func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

func ipToUint32(ipAddr string) (uint32, error) {
	ip := net.ParseIP(ipAddr)
	if ip == nil {
		return 0, errors.New("wrong ipAddr format")
	}
	ip = ip.To4()
	return binary.BigEndian.Uint32(ip), nil
}

func uint32ToIP(ipLong uint32) string {
	ipByte := make([]byte, 4)
	binary.BigEndian.PutUint32(ipByte, ipLong)
	ip := net.IP(ipByte)
	return ip.String()
}
