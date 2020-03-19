package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"regexp"
	"syscall"
	"time"

	"github.com/go-redis/redis"
	"github.com/joho/godotenv"
	"github.com/jxsl13/twapi/econ"
)

var (
	config          = Config{}
	playerJoinRegex = regexp.MustCompile(`player is ready\. ClientID=([\d]+) addr=[^\d]{0,2}([\d]{1,3}\.[\d]{1,3}\.[\d]{1,3}\.[\d]{1,3})[^\d]{0,2}`)
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

func parseFileAndAddIPsToCache(filename string) (int, error) {
	file, err := os.Open(filename)
	if err != nil {
		return 0, err
	}
	r := redis.NewClient(&redis.Options{
		Addr:     string(config.RedisAddress),
		Password: string(config.RedisPassword),
	})
	defer r.Close()

	foundIPs := 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {

		ips := parseIPLine(scanner.Text())
		foundIPs += len(ips)

		for _, ip := range ips {
			r.Set(ip, true, 0) // Add VPN IP to cache.
		}
	}
	return foundIPs, nil
}

func parseFileAndRemoveIPsFromCache(filename string) (int, error) {
	file, err := os.Open(filename)
	if err != nil {
		return 0, err
	}
	r := redis.NewClient(&redis.Options{
		Addr:     string(config.RedisAddress),
		Password: string(config.RedisPassword),
	})
	defer r.Close()

	foundIPs := 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {

		ips := parseIPLine(scanner.Text())
		foundIPs += len(ips)

		for _, ip := range ips {
			r.Del(ip)
		}
	}
	return foundIPs, nil
}

func main() {

	addFile := ""
	removeFile := ""
	flag.StringVar(&addFile, "add", "", "pass a text file with IPs and IP subnets to be added to the database")
	flag.StringVar(&removeFile, "remove", "", "pass a text file with IPs and IP subnets to be removed from the database")
	flag.Parse()

	// If flag passed, add parsed ips to database.
	if addFile != "" {
		foundIPs, err := parseFileAndAddIPsToCache(addFile)
		if err != nil {
			fmt.Println(err)
		}
		fmt.Printf("Added %d VPN IPs to the redis cache.", foundIPs)
		return
	}

	if removeFile != "" {
		foundIPs, err := parseFileAndRemoveIPsFromCache(removeFile)
		if err != nil {
			fmt.Println(err)
		}
		fmt.Printf("Removed %d IPs from the redis cache.", foundIPs)
		return
	}

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
