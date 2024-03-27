package econ

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"sync"
	"time"

	"github.com/jxsl13/TeeworldsEconVPNDetection/vpn"
	"github.com/jxsl13/twapi/econ"
)

var (
	// 0: full 1: ID 2: IP
	ddnetJoinRegex = regexp.MustCompile(`(?i)player has entered the game\. ClientID=([\d]+) addr=[^\d]{0,2}([\d]{1,3}\.[\d]{1,3}\.[\d]{1,3}\.[\d]{1,3})[^\d]{0,2}`)

	// 0: full 1: ID 2: IP 3: port 4: version 5: name 6: clan 7: country
	playerzCatchJoinRegex = regexp.MustCompile(`(?i)id=([\d]+) addr=([a-fA-F0-9\.\:\[\]]+):([\d]+) version=(\d+) name='(.{0,20})' clan='(.{0,16})' country=([-\d]+)$`)

	// 0: full 1: ID 2: IP
	playerVanillaJoinRegex = regexp.MustCompile(`(?i)player is ready\. ClientID=([\d]+) addr=[^\d]{0,2}([\d]{1,3}\.[\d]{1,3}\.[\d]{1,3}\.[\d]{1,3})[^\d]{0,2}`)
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
		log.Println(err)
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
	startedWG *sync.WaitGroup,
	stoppedWG *sync.WaitGroup,
) {
	defer stoppedWG.Done()

	var once sync.Once
	defer func() {
		once.Do(func() {
			startedWG.Done()
		})
	}()

	log.Printf("Dialing to %s\n", addr)
	econ, err := econ.DialTo(addr, pw)
	if err != nil {
		log.Printf("Could not connect to %s, error: %v\n", addr, err)
		return
	}
	go func(addr string) {
		<-ctx.Done()
		_ = econ.Close()
	}(addr)

	accumulatedRetryTime := time.Duration(0)
	retries := 0
	var matches []string
	var ip string

	for {
		if retries == 0 {
			log.Println("Connected to server:", addr)

			const logCommand = "ec_output_level 2"
			// enable verbose logging which is required for the join messages
			log.Printf("Setting: %q for connection %s\n", logCommand, addr)
			err = econ.WriteLine(logCommand)
			if err != nil {
				log.Printf("Failed to set %q for connection %s: %v\n", logCommand, addr, err)
				return
			}
			once.Do(func() { startedWG.Done() })
		} else {
			log.Println("Retrying to connect to server:", addr)
		}

	parseLine:
		for {
			line, err := econ.ReadLine()
			if err != nil {
				select {
				case <-ctx.Done():
					log.Printf("Closing connection to: %s\n", addr)
					return
				default:
					log.Printf("Lost connection to %s, error: %v\n", addr, err)
					break parseLine
				}
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
			log.Printf("%s joined server %s\n", ip, addr)
			go vpnCheck(
				econ,
				ip,
				checker,
				vpnBantime,
				vpnBanReason,
			)

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
