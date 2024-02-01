package econ

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"time"

	"github.com/jxsl13/TeeworldsEconVPNDetection/vpn"
	"github.com/jxsl13/twapi/econ"
)

var (
	// 0: full 1: ID 2: IP
	ddnetJoinRegex = regexp.MustCompile(`player has entered the game\. ClientID=([\d]+) addr=[^\d]{0,2}([\d]{1,3}\.[\d]{1,3}\.[\d]{1,3}\.[\d]{1,3})[^\d]{0,2}`)

	// 0: full 1: ID 2: IP 3: port 4: version 5: name 6: clan 7: country
	playerzCatchJoinRegex = regexp.MustCompile(`id=([\d]+) addr=([a-fA-F0-9\.\:\[\]]+):([\d]+) version=(\d+) name='(.{0,20})' clan='(.{0,16})' country=([-\d]+)$`)

	// 0: full 1: ID 2: IP
	playerVanillaJoinRegex = regexp.MustCompile(`player is ready\. ClientID=([\d]+) addr=[^\d]{0,2}([\d]{1,3}\.[\d]{1,3}\.[\d]{1,3}\.[\d]{1,3})[^\d]{0,2}`)
)

func vpnCheck(
	econ *econ.Conn,
	ip string,
	checker *vpn.VPNChecker,
	vpnBantime time.Duration,
	vpnBanReason string,
) {

	isVPN, reason, err := checker.IsVPN(ip)
	if err != nil {
		log.Println(err.Error())
		return
	}

	// vpn is saved as 1, banserver bans as text
	if isVPN {
		tag := "[banserver] :" // manually added with custom reason
		minutes := int(vpnBantime.Minutes())

		if reason == "" {
			tag = "[is a vpn]  :" //
			reason = vpnBanReason
		}

		_ = econ.WriteLine(fmt.Sprintf("ban %s %d %s", ip, minutes, reason))
		log.Println(tag, ip, "(", reason, ")")
	} else {
		log.Println("[clean ip]: ", ip)
	}
}

func NewEvaluationRoutine(
	ctx context.Context,
	addr string,
	pw string,
	checker *vpn.VPNChecker,
	reconnDelay time.Duration,
	reconnTimeout time.Duration,
	vpnBantime time.Duration,
	vpnBanReason string,
) {

	econ, err := econ.DialTo(addr, pw)
	if err != nil {
		checker.Close()
		log.Fatalf("Could not connect to %s, error: %s\n", addr, err.Error())
		return
	}
	defer econ.Close()

	accumulatedRetryTime := time.Duration(0)
	retries := 0
	var matches []string
	var ip string

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

				// TODO: check if it's a join message synchronously
				if matches = ddnetJoinRegex.FindStringSubmatch(line); len(matches) > 0 {
					ip = matches[2]
				} else if matches = playerzCatchJoinRegex.FindStringSubmatch(line); len(matches) > 0 {
					ip = matches[2]
				} else if matches = playerVanillaJoinRegex.FindStringSubmatch(line); len(matches) > 0 {
					ip = matches[2]
				} else {
					continue
				}
				go vpnCheck(
					econ,
					ip,
					checker,
					vpnBantime,
					vpnBanReason,
				)
			}

		}

		select {
		case <-ctx.Done():
			log.Printf("Closing connection to: %s\n", addr)
			return
		case <-time.After(reconnDelay):
			accumulatedRetryTime += reconnDelay

			// if we exceed a threshold, stop the goroutine
			if accumulatedRetryTime > reconnTimeout {
				log.Println("Exceeded reconnect timeout, stopping routine:", addr)
				return
			}
			retries++
		}
	}
}
