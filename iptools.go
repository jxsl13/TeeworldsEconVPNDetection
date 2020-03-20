package main

import (
	"net"
	"regexp"
)

var (
	ipv4SubnetWithReasonRegex = regexp.MustCompile(`([\d]{1,3}\.[\d]{1,3}\.[\d]{1,3}\.[\d]{1,3}\/[\d]{1,2})([\s]*#[\s]*(.*))`)
	ipv4SubnetRegex           = regexp.MustCompile(`[\d]{1,3}\.[\d]{1,3}\.[\d]{1,3}\.[\d]{1,3}\/[\d]{1,2}`)
	ipv4WithReasonRegex       = regexp.MustCompile(`([\d]{1,3}\.[\d]{1,3}\.[\d]{1,3}\.[\d]{1,3})([\s]*#[\s]*(.*))`)
	ipv4Regex                 = regexp.MustCompile(`[\d]{1,3}\.[\d]{1,3}\.[\d]{1,3}\.[\d]{1,3}`)
)

// reason 1 -> use default ban reason, reason non digit -> use this as reason
// reason 0 -> don't ban
func parseIPLine(line string) (ipsWithReasons map[string]string) {
	// 0.0.0.0 -> len = 7
	if len(line) < 7 {
		return nil
	}

	match := ipv4SubnetWithReasonRegex.FindStringSubmatch(line)
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
		reason := "1"
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
		const reason = "1"
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

func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}
