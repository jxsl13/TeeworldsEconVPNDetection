package main

import (
	"errors"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/go-redis/redis"
)

// NewVPNChecker creates a new checker that can be asked for VPN IPs.
// it connects to the redis database for caching and requests information from all existing
// API endpoints that provode free VPN detections.
func NewVPNChecker(cfg *Config) *VPNChecker {

	apis := []VPN{}
	if !cfg.Offline {
		// share client with all apis
		httpClient := &http.Client{}
		apis = []VPN{
			NewGetIPIntelNet(httpClient, cfg.Email, 0.95),
			NewIPHub(httpClient, cfg.IPHubToken),
			NewIPTeohIO(httpClient),
		}
	}

	return &VPNChecker{
		redis.NewClient(
			&redis.Options{
				Addr:     string(cfg.RedisAddress),
				Password: string(cfg.RedisPassword),
			}),
		apis,
		cfg.Offline,
	}
}

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
	Apis    []VPN
	Offline bool
}

//
func (rdb *VPNChecker) foundInCache(sIP string) (found bool, isVPN bool) {
	found = false
	isVPN = false

	sIsVPN, err := rdb.Get(sIP).Result()
	if err != nil {
		return
	}

	found = true
	isVPN = sIsVPN == "1"
	return
}

func (rdb *VPNChecker) foundOnline(sIP string) (IsVPN bool) {

	results := make([]Valid, len(rdb.Apis))

	for idx, api := range rdb.Apis {

		isVPNTmp, err := api.IsVPN(sIP)

		if err != nil {
			log.Println("[ERROR]:", api.String(), ":", err.Error())
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
		log.Printf("[in cache]: %s", IP)
		return isCacheVPN, nil
	}

	if rdb.Offline {
		// if the detection is offline, cache only,
		// caching of default no values makes no sense, so no caching here.
		return false, nil
	}

	isOnlineVPN := rdb.foundOnline(IP)
	log.Printf("[online]:  %s", IP)
	// update cache values
	if isOnlineVPN {
		// forever vpn
		rdb.Set(IP, true, 0)
	} else {
		// for one week no vpn
		rdb.Set(IP, false, 24*7*time.Hour)
	}

	return isOnlineVPN, nil
}
