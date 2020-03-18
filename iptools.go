package main

import (
	"net"
	"regexp"
)

var (
	ipv4SubnetRegex = regexp.MustCompile(`([\d]{1,3}\.[\d]{1,3}\.[\d]{1,3}\.[\d]{1,3})(\/[\d]{1,2})?`)
)

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

func parseIPLine(line string) (ips []string) {
	match := ipv4SubnetRegex.FindStringSubmatch(line)

	switch len(match) {
	case 1 + 1:
		ips = make([]string, 1)
		ips[0] = match[1]
	case 1 + 2:
		ipList, err := ipsFromCIDR(match[0])
		if err != nil {
			return nil
		}
		return ipList
	}

	return nil

}
