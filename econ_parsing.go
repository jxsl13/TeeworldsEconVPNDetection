package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jxsl13/twapi/econ"
)

func parseLine(econ *econ.Conn, checker *VPNChecker, line string) {
	var matches []string

	switch config.ZCatchLogFormat {
	case true:
		matches = playerzCatchJoinRegex.FindStringSubmatch(line)
	default:
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
		tag := "[banserver]:" // manually added with custom reason
		ID := matches[1]
		minutes := int(config.VPNBanTime.Minutes())

		if reason == "" {
			tag = "[is a vpn] :" //
			reason = config.VPNBanReason
		}

		econ.WriteLine(fmt.Sprintf("ban %s %d %s", ID, minutes, reason))
		log.Println(tag, IP, "(", reason, ")")
		return
	}

	log.Println("[clean IP]:", IP)

}

func econEvaluationRoutine(ctx context.Context, checker *VPNChecker, addr string, pw string) {

	econ, err := econ.DialTo(addr, pw)
	if err != nil {
		log.Printf("Could not connect to %s, error: %s\n", addr, err.Error())
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
				go parseLine(econ, checker, line)
			}

		}

		select {
		case <-ctx.Done():
			log.Printf("Closing connection to: %s\n", addr)
			return
		case <-time.After(config.ReconnectDelay):
			accumulatedRetryTime += config.ReconnectDelay

			// if we exceed a threshold, stop the goroutine
			if accumulatedRetryTime > config.ReconnectTimeout {
				log.Println("Exceeded reconnect timeout, stopping routine:", addr)
				return
			}
			retries++
		}
	}
}
