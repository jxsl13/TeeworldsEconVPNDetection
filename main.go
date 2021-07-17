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
)

var (
	config *Config

	// 0: full 1: ID 2: IP
	playerVanillaJoinRegex = regexp.MustCompile(`player is ready\. ClientID=([\d]+) addr=[^\d]{0,2}([\d]{1,3}\.[\d]{1,3}\.[\d]{1,3}\.[\d]{1,3})[^\d]{0,2}`)

	// 0: full 1: ID 2: IP 3: port 4: version 5: name 6: clan 7: country
	playerzCatchJoinRegex = regexp.MustCompile(`id=([\d]+) addr=([a-fA-F0-9\.\:\[\]]+):([\d]+) version=(\d+) name='(.{0,20})' clan='(.{0,16})' country=([-\d]+)$`)
)

func init() {
	var err error
	config, err = NewConfig(".env")
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}

func main() {

	addFile := ""
	removeFile := ""
	flag.StringVar(&addFile, "add", "", "pass a text file with IPs and IP subnets to be added to the database")
	flag.StringVar(&removeFile, "remove", "", "pass a text file with IPs and IP subnets to be removed from the database")
	flag.BoolVar(&config.Offline, "offline", false, "do not use the api endpoints, only rely on the cache")
	flag.Parse()

	// If flag passed, add parsed ips to database.
	if addFile != "" {
		foundIPs, err := parseFileAndAddIPsToCache(addFile)
		if err != nil {
			fmt.Println(err)
		}
		fmt.Printf("Added %d VPN IP ranges to the redis(DB: %d).\n", foundIPs, config.RedisDB)
		return
	}

	if removeFile != "" {
		foundIPs, err := parseFileAndRemoveIPsFromCache(removeFile)
		if err != nil {
			fmt.Println(err)
		}
		fmt.Printf("Removed %d IP ranges from the redis(DB: %d).\n", foundIPs, config.RedisDB)
		return
	}

	if config.Offline {
		log.Println("The detection is running in offline mode, only cached IPs are banned.")
	}

	ctx, cancel := context.WithCancel(context.Background())
	checker, err := NewVPNChecker(config)
	if err != nil {
		log.Fatalln(err)
	}
	defer checker.Close()

	// start goroutines
	for idx, addr := range config.EconServers {
		go econEvaluationRoutine(ctx, checker, addr, config.EconPasswords[idx])
	}

	// block main goroutine until the application receives a signal
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM)
	<-sc
	cancel()
	log.Println("Shutting down...")
}
