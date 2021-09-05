package config

import (
	"bufio"
	"fmt"
	"os"

	"github.com/jxsl13/goripr"
)

func parseFileAndAddIPsToCache(filename, redisAddress, redisPassword string, redisDB int) (int, error) {
	file, err := os.Open(filename)
	if err != nil {
		return 0, err
	}

	r, err := goripr.NewClient(goripr.Options{
		Addr:     redisAddress,
		Password: redisPassword,
		DB:       redisDB,
	})
	if err != nil {
		return 0, err
	}
	defer r.Close()

	foundIpRanges := 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		ip, reason, err := parseIPLine(line)
		if err != nil {
			continue
		}
		err = r.Insert(ip, reason)
		if err != nil {
			return 0, fmt.Errorf("%w: %s", err, line)
		}

		// counter
		foundIpRanges++
	}
	return foundIpRanges, nil
}
