package cmd

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/jxsl13/goripr/v2"
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

func parseFileAndAddIPsToCache(ctx context.Context, r *goripr.Client, filename string) (int, error) {
	file, err := os.Open(filename)
	if err != nil {
		return 0, err
	}

	foundIpRanges := 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		ip, reason, err := parseIPLine(line)
		if err != nil {
			continue
		}
		err = r.Insert(ctx, ip, reason)
		if err != nil {
			return 0, fmt.Errorf("%w: %s", err, line)
		}

		// counter
		foundIpRanges++
	}
	return foundIpRanges, nil
}

func parseFileAndRemoveIPsFromCache(ctx context.Context, r *goripr.Client, filename string) (int, error) {
	file, err := os.Open(filename)
	if err != nil {
		return 0, err
	}

	foundRanges := 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		ip, _, err := parseIPLine(line)
		if err != nil {
			continue
		}
		err = r.Remove(ctx, ip)
		if err != nil {
			return 0, err
		}

		foundRanges++
	}

	return foundRanges, nil
}
