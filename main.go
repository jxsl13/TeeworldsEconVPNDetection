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

		isVPN, reason, err := checker.IsVPN(IP)
		if err != nil {
			log.Println(err.Error())
			return
		}

		// vpn is saved as 1, banserver bans as text
		if isVPN && reason == "1" {
			reason = config.VPNBanReason

			minutes := int(config.VPNBanTime.Minutes())
			econ.WriteLine(fmt.Sprintf("ban %s %d %s", ID, minutes, reason))
			log.Println("[is a vpn] :", IP, "(", reason, ")")
			return
		}

		if isVPN {
			minutes := int(config.VPNBanTime.Minutes())
			econ.WriteLine(fmt.Sprintf("ban %s %d %s", ID, minutes, reason))
			log.Println("[banserver]:", IP, "(", reason, ")")
			return
		}
		log.Println("[valid]:", IP)
	}
}

func econEvaluationRoutine(ctx context.Context, checker *VPNChecker, addr address, pw password) {

	econ, err := econ.DialTo(string(addr), string(pw))
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

func parseFileAndAddIPsToCache(filename string) (int, error) {
	file, err := os.Open(filename)
	if err != nil {
		return 0, err
	}
	options := config.RedisOptions
	r := redis.NewClient(&options)
	defer r.Close()

	foundIPs := 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {

		ips := parseIPLine(scanner.Text(), "1")
		foundIPs += len(ips)

		transaction := r.TxPipeline()
		for ip, reason := range ips {
			// default reason = "1"
			// custom reason = "text"
			transaction.Set(ip, reason, 0) // Add IP to cache.
		}
		transaction.Exec()
	}
	return foundIPs, nil
}

func parseFileAndRemoveIPsFromCache(filename string) (int, error) {
	file, err := os.Open(filename)
	if err != nil {
		return 0, err
	}
	options := config.RedisOptions
	r := redis.NewClient(&options)
	defer r.Close()

	foundIPs := 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {

		ips := parseIPLine(scanner.Text(), "")
		foundIPs += len(ips)

		transaction := r.TxPipeline()
		for ip := range ips {
			transaction.Del(ip)
		}
		transaction.Exec()
	}
	return foundIPs, nil
}

func parseFileAndWhiteListInCache(filename string) (int, error) {
	file, err := os.Open(filename)
	if err != nil {
		return 0, err
	}
	options := config.RedisOptions
	r := redis.NewClient(&options)
	defer r.Close()

	foundIPs := 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {

		ips := parseIPLine(scanner.Text(), "0")
		foundIPs += len(ips)

		transaction := r.TxPipeline()
		for ip := range ips {
			transaction.Set(ip, "0", 0) // Force whitelisting in cache
		}
		transaction.Exec()
	}
	return foundIPs, nil
}

func main() {

	addFile := ""
	removeFile := ""
	whitelistFile := ""
	flag.StringVar(&addFile, "add", "", "pass a text file with IPs and IP subnets to be added to the database")
	flag.StringVar(&removeFile, "remove", "", "pass a text file with IPs and IP subnets to be removed from the database")
	flag.StringVar(&whitelistFile, "whitelist", "", "whitelist these IPs forever in cache, meaning they will never be banned.")
	flag.BoolVar(&config.Offline, "offline", false, "do not use the api endpoints, only rely on the cache")
	flag.Parse()

	// If flag passed, add parsed ips to database.
	if addFile != "" {
		foundIPs, err := parseFileAndAddIPsToCache(addFile)
		if err != nil {
			fmt.Println(err)
		}
		fmt.Printf("Added %d VPN IPs to the redis(DB:%d) cache.\n", foundIPs, config.RedisOptions.DB)
		return
	}

	if removeFile != "" {
		foundIPs, err := parseFileAndRemoveIPsFromCache(removeFile)
		if err != nil {
			fmt.Println(err)
		}
		fmt.Printf("Removed %d IPs from the redis(DB:%d) cache.\n", foundIPs, config.RedisOptions.DB)
		return
	}

	if whitelistFile != "" {
		foundIPs, err := parseFileAndWhiteListInCache(whitelistFile)
		if err != nil {
			fmt.Println(err)
		}
		fmt.Printf("Whitelisted %d IPs in the redis(DB:%d) cache.\n", foundIPs, config.RedisOptions.DB)
		return
	}

	if config.Offline {
		log.Println("The detection is running in offline mode, only cached IPs are banned.")
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
