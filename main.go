package main

import (
	"errors"
	"log"
	"net"
	"net/http"
	"strings"
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

	percentage := trueValue / total

	IsVPN = percentage >= 0.5
	return
}

// IsVPN checks firstly in cache and then online.
func (rdb *VPNChecker) IsVPN(sIP string) (bool, error) {

	IP := net.ParseIP(sIP).To4().String()
	log.Println(IP)
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
	//args := os.Args[1:]
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
	}

	// share client with all apis
	httpClient := &http.Client{}

	// list of posisble apis to use
	apis := []VpnAPI{
		GetIPIntelNet{httpClient, CooldownHandler{}, Email, 0.9},
		IPHub{httpClient, CooldownHandler{}, IPHubToken},
		IPTeohIO{httpClient, CooldownHandler{}},
	}

	rdb := VPNChecker{redis.NewClient(&redis.Options{
		Addr:     RedisAddress,
		Password: RedisPassword, // no password set
		DB:       0,             // use default DB
	}),
		apis}
	defer rdb.Close() // close before return

	CurrentServer := GameServerAddresses[0]
	CurrentPassword := Passwords[0]

	caller := internalStandardCaller{CurrentPassword, rdb, env}
	err = telnet.DialToAndCall(CurrentServer, caller)
	if err != nil {
		log.Println("Failed to connect to the remote server:", CurrentServer, err.Error())
		return
	}

}
