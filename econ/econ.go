package econ

import (
	"fmt"
	"log"
	"regexp"
	"time"

	"github.com/jxsl13/TeeworldsEconVPNDetectionGo/config"
	"github.com/jxsl13/TeeworldsEconVPNDetectionGo/servers"
	"github.com/jxsl13/twapi/econ"
)

var (

	// 0: full 1: ID 2: IP
	playerVanillaJoinRegex = regexp.MustCompile(`player is ready\. ClientID=([\d]+) addr=[^\d]{0,2}([\d]{1,3}\.[\d]{1,3}\.[\d]{1,3}\.[\d]{1,3})[^\d]{0,2}`)

	// 0: full 1: ID 2: IP 3: port 4: version 5: name 6: clan 7: country
	playerzCatchJoinRegex = regexp.MustCompile(`id=([\d]+) addr=([a-fA-F0-9\.\:\[\]]+):([\d]+) version=(\d+) name='(.{0,20})' clan='(.{0,16})' country=([-\d]+)$`)
)

func parseLine(econ *econ.Conn, line string) {
	var matches []string

	cfg := config.New()
	checker := cfg.Checker()

	switch cfg.LogFormat {
	case "zCatch":
		matches = playerzCatchJoinRegex.FindStringSubmatch(line)
	case "Vanilla":
		fallthrough // fallthrough to default
	default:
		// defaults to Vanilla
		matches = playerVanillaJoinRegex.FindStringSubmatch(line)
	}

	if len(matches) <= 0 {
		return
	}

	IP := matches[2]

	isVPN, reason, err := checker.IsVPN(IP)
	if err != nil {
		log.Println(err.Error())
		return
	}

	// vpn is saved as 1, banserver bans as text
	if isVPN {
		tag := "[banserver] :" // manually added with custom reason
		minutes := int(cfg.VPNBanTime.Minutes())

		if reason == "" {
			tag = "[is a vpn]  :" //
			reason = cfg.VPNBanReason
		}

		econ.WriteLine(fmt.Sprintf("ban %s %d %s", IP, minutes, reason))
		log.Println(tag, IP, "(", reason, ")")
	} else if servers.IsTeeworldsServer(IP) {
		tag := "[is a proxy]:"
		minutes := int(cfg.ProxyBanDuration.Minutes())
		reason := cfg.ProxyBanReason

		econ.WriteLine(fmt.Sprintf("ban %s %d %s", IP, minutes, reason))
		log.Println(tag, IP, "(", reason, ")")
	} else {
		log.Println("[clean IP] :", IP)
	}
}

func NewEvaluationRoutine(addr string, pw string) {
	cfg := config.New()
	checker := cfg.Checker()
	ctx := cfg.Context()

	econ, err := econ.DialTo(addr, pw)
	if err != nil {
		checker.Close()
		log.Fatalf("Could not connect to %s, error: %s\n", addr, err.Error())
		return
	}
	defer econ.Close()

	accumulatedRetryTime := time.Duration(0)
	retries := 0

	for {
		if retries == 0 {
			log.Println("Connected to server:", addr)
		} else {
			log.Println("Retrying to connect to server:", addr)
		}

	parseLine:
		for {
			select {
			case <-ctx.Done():
				log.Printf("Closing connection to: %s\n", addr)
				return
			default:
				line, err := econ.ReadLine()
				if err != nil {
					log.Printf("Lost connection to %s, error: %s\n", addr, err.Error())
					break parseLine
				}
				go parseLine(econ, line)
			}

		}

		select {
		case <-ctx.Done():
			log.Printf("Closing connection to: %s\n", addr)
			return
		case <-time.After(cfg.ReconnectDelay):
			accumulatedRetryTime += cfg.ReconnectDelay

			// if we exceed a threshold, stop the goroutine
			if accumulatedRetryTime > cfg.ReconnectTimeout {
				log.Println("Exceeded reconnect timeout, stopping routine:", addr)
				return
			}
			retries++
		}
	}
}
