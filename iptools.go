package main

import (
	"errors"
	"regexp"
	"strings"
)

var (
	splitRegex = regexp.MustCompile(`^\s*([0-9\.\-\/]+)\s*(#\s*(.*[^\s])\s*)?$`)
)

func parseIPLine(line string) (ipRange, reason string, err error) {
	matches := splitRegex.FindStringSubmatch(line)
	if len(matches) == 0 {
		return "", "", errors.New("empty")
	}
	return strings.TrimSpace(matches[1]), strings.TrimSpace(matches[3]), nil
}
