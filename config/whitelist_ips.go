package config

import (
	"bufio"
	"log"
	"os"

	"github.com/jxsl13/goripr"
)

func parseFileAndRemoveIPsFromCache(filename, redisAddress, redisPassword string, redisDB int) (int, error) {
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

	foundRanges := 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		ip, _, err := parseIPLine(line)
		if err != nil {
			log.Println("Skipped line: ", line)
			continue
		}
		err = r.Remove(ip)
		if err != nil {
			return 0, err
		}

		foundRanges++
	}

	return foundRanges, nil
}
