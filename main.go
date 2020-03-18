package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"regexp"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/jxsl13/twapi/econ"
)

var (
	config          = Config{}
	playerJoinRegex = regexp.MustCompile(`player is ready\. ClientID=([\d]+) addr=([\d]{1,3}\.[\d]{1,3}\.[\d]{1,3}\.[\d]{1,3})`)
)

func init() {

	var env map[string]string
	env, err := godotenv.Read(".env")

	if err != nil {
		log.Println("Error parsing '.env' file:", err.Error())
		return
	}

	config, err = NewConfig(env)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}

func parseLine(econ *econ.Conn, checker *VPNChecker, line string) {

	matches := playerJoinRegex.FindStringSubmatch(line)
	if len(matches) > 0 {
		ID := matches[1]
		IP := matches[2]

		isVPN, err := checker.IsVPN(IP)
		if err != nil {
			log.Println(err.Error())
			return
		}

		if isVPN {
			minutes := int(config.VPNBanTime.Minutes())
			econ.WriteLine(fmt.Sprintf("ban %s %d %s", ID, minutes, config.VPNBanReason))
			log.Println("[is a vpn] :", IP)
		} else {
			log.Println("[not a vpn]:", IP)
		}
	}
}

func econEvaluationRoutine(ctx context.Context, checker *VPNChecker, addr address, pw password) {

	econ, err := econ.DialTo(string(addr), string(pw))
	if err != nil {
		log.Printf("Could not connect to %s, error: %s", addr, err.Error())
		return
	}
	defer econ.Close()

	reconnectTimer := time.Second
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
				log.Printf("Closing connection to %s", addr)
				return
			default:
				line, err := econ.ReadLine()
				if err != nil {
					log.Printf("Lost connection to %s, error: %s", addr, err.Error())
					break parseLine
				}
				go parseLine(econ, checker, line)
			}

		}

		select {
		case <-ctx.Done():
			log.Println("Closing connection to", addr)
			return
		case <-time.After(reconnectTimer):
			// sleep before retrying
			time.Sleep(reconnectTimer)

			// double timer on each attempt
			reconnectTimer *= 2

			// if we exceed a threshold, stop the goroutine
			if reconnectTimer > config.ReconnectTimeout {
				log.Println("Exceeded reconnect timeout, stopping routine:", addr)
				return
			}
			retries++
		}
	}

}

func main() {

	textFile := ""
	flag.StringVar(&textFile, "f", "", "pass a text file with IPs and IP subnets")

	ctx, cancel := context.WithCancel(context.Background())
	checker := NewVPNChecker(&config)

	// start goroutines
	for idx, addr := range config.EconServers {
		go econEvaluationRoutine(ctx, checker, addr, config.EconPasswords[idx])
	}

	// block main goroutine until the application receives a signal
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc
	cancel()
	log.Println("Shutting down...")

}
