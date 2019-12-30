package main

import (
	"errors"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis"
	"github.com/joho/godotenv"
	"github.com/reiver/go-telnet"
)

// Valid is used to represent the answer of an api endpoint
type Valid struct {
	IsValid bool
	Value   bool
}

// VPNChecker encapsulates the redis database as cache and the
// implemented api endpoints in order to determine, whether an ip is a vpn based on
// either the caching information or based on the implemented api endpoints, whether
// an ip is a vpn.
type VPNChecker struct {
	*redis.Client
	Apis []VpnAPI
}

//
func (rdb *VPNChecker) foundInCache(sIP string) (found bool, isVPN bool) {
	found = false
	isVPN = false

	rResult, rErr := rdb.Ping().Result()

	if rErr != nil && rResult != "PONG" {
		log.Println("[redis]: could not connect to the redis database, caching disabled!")
		return
	}

	sIsVPN, err := rdb.Get(sIP).Result()
	if err != nil {
		return
	}

	found = true
	isVPN = sIsVPN == "1"
	return
}

func (rdb *VPNChecker) foundOnline(sIP string) (IsVPN bool) {
	IsVPN = false

	results := make([]Valid, len(rdb.Apis))

	for idx, api := range rdb.Apis {

		isVPNTmp, err := api.IsVpn(sIP)

		if err != nil {
			log.Println("[ERROR]:", err.Error())
			results[idx] = Valid{false, false}
			continue
		}

		results[idx] = Valid{true, isVPNTmp}
	}

	total := 0.0
	trueValue := 0.0
	for _, valid := range results {
		if valid.IsValid {
			total += 1.0
			if valid.Value {
				trueValue += 1.0
			}
		}
	}

	if total == 0.0 {
		log.Println("[ERROR]: All APIs seem to have exceeded their rate limitations.")
		IsVPN = false
		return
	}
	percentage := trueValue / total

	IsVPN = percentage >= 0.6
	return
}

// IsVPN checks firstly in cache and then online.
func (rdb *VPNChecker) IsVPN(sIP string) (bool, error) {

	IP := net.ParseIP(sIP).To4().String()
	if IP == "<nil>" {
		return false, errors.New("Invalid IP passed, expexted IPv4")
	}

	found, isCacheVPN := rdb.foundInCache(IP)

	if found {
		return isCacheVPN, nil
	}

	isOnlineVPN := rdb.foundOnline(IP)

	pong, err := rdb.Ping().Result()

	if err == nil && pong == "PONG" {
		if isOnlineVPN {
			// forever vpn
			rdb.Set(IP, true, 0)
		} else {
			// for one week no vpn
			rdb.Set(IP, false, 24*7*time.Hour)
		}
	}

	return isOnlineVPN, nil
}

func main() {
	var env map[string]string
	env, err := godotenv.Read(".env")

	if err != nil {
		log.Println("Error parsing '.env' file:", err.Error())
		return
	}

	// retrieved from .env file
	IPHubToken := env["IPHUB_TOKEN"]
	Email := env["EMAIL"]
	RedisAddress := env["REDIS_ADDRESS"]
	RedisPassword := env["REDIS_PASSWORD"]
	GameServerAddresses := strings.Split(env["SERVER_LIST"], " ")
	Passwords := strings.Split(env["PASSWORDS"], " ")
	ReconnectTimeoutMinutes, err := strconv.Atoi(env["RECONNECT_TIMEOUT_MINS"])
	if err != nil || ReconnectTimeoutMinutes <= 0 {
		ReconnectTimeoutMinutes = 60
	}

	if RedisAddress == "" {
		RedisAddress = "localhost:6379"
	}
	if len(GameServerAddresses) == 0 {
		log.Println("Please add a 'SERVER_LIST=127.0.0.1:1234 127.0.0.1:5678 ...' to your .env file!")
		return
	}
	if len(Passwords) == 0 {
		log.Println("Please add a 'PASSWORDS=12345 PA$$W0RD ...' to your .env file!")
		return
	}
	if len(GameServerAddresses) != len(Passwords) && len(Passwords) != 1 {
		log.Println("Number of server addresses and passwords don't match. Either use one password for all servers or one password per server.")
		return
	} else if len(Passwords) == 1 {
		// have as many passwords as server addresses
		for len(Passwords) < len(GameServerAddresses) {
			Passwords = append(Passwords, Passwords[0])
		}
	}

	// share client with all apis
	httpClient := &http.Client{}

	dailyRequestLimits := []int{
		500,  // 500 api calls per day - GetIPIntel
		1000, // 1000 api calls per day - IPHub
		1000, // 1000 api calls per day - ip.teoh.io
	}

	var dailytLimits []*RateLimiter
	for _, dailyRequestLimit := range dailyRequestLimits {
		dailytLimits = append(dailytLimits, NewRateLimiter(24*time.Hour, dailyRequestLimit))
	}

	// list of posisble apis to use
	apis := []VpnAPI{
		&GetIPIntelNet{httpClient, dailytLimits[0], Email, 0.9}, // 500 api calls per day
		&IPHub{httpClient, dailytLimits[1], IPHubToken},         // 1000 api calls per day
		&IPTeohIO{httpClient, dailytLimits[2]},                  // 1000  api calls per day
	}

	rdb := VPNChecker{redis.NewClient(&redis.Options{
		Addr:     RedisAddress,
		Password: RedisPassword,
		DB:       0, // use default DB
	}),
		apis}
	defer rdb.Close() // close before return

	// block main thread until goroutines finish execution
	var wg sync.WaitGroup

	// wraps the call in order to handle errors
	Run := func(CurrentServer string, CurrentPassword string) {
		// call on return
		defer wg.Done()

		reconnectTimer := time.Second
		retries := 0
		for {
			if retries == 0 {
				log.Println("Connecting to server:", CurrentServer)
			} else {
				log.Println("Retrying to connect to server:", CurrentServer)
			}

			err := telnet.DialToAndCall(CurrentServer, internalStandardCaller{CurrentServer, CurrentPassword, rdb, env, &wg})
			if err != nil {
				log.Println("Could not connect to server:", CurrentServer)

				// sleep before retrying
				time.Sleep(reconnectTimer)

				// double timer on each attempt
				reconnectTimer *= 2

				// if we exceed a threshold, stop the goroutine
				if reconnectTimer > time.Minute*time.Duration(ReconnectTimeoutMinutes) {
					log.Println("Exceeded reconnect timeout, stopping routine:", CurrentServer)
					break
				}
			}
			retries++
		}
	}

	wg.Add(len(GameServerAddresses))
	// start goroutines
	for idx, CurrentServer := range GameServerAddresses {
		go Run(CurrentServer, Passwords[idx])
		time.Sleep(1 * time.Second)
	}
	wg.Wait()
}
