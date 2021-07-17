package main

import (
	"errors"
	"regexp"
	"strings"
)

var (
	splitRegex = regexp.MustCompile(`^\s*([\s0-9\.\-\/]+)\s*(#\s*(.*[^\s])\s*)?$`)
)

func parseIPLine(line string) (ipRange, reason string, err error) {
	matches := splitRegex.FindStringSubmatch(line)
	if len(matches) == 0 {
		return "", "", errors.New("empty")
	}

	ipRange = strings.TrimSpace(matches[1])
	reason = strings.TrimSpace(matches[3])

	ips := strings.Split(ipRange, "-")
	for idx, ip := range ips {
		ips[idx] = strings.TrimSpace(ip)
	}
	ipRange = strings.Join(ips, "-")

	return ipRange, reason, nil
}
