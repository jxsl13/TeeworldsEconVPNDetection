package main

import (
	"errors"
	"log"
	"net"
	"net/http"

	"github.com/jxsl13/goripr"
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
	r       *goripr.Client
	Apis    []VPN
	Offline bool
}

func (rdb *VPNChecker) Close() error {
	return rdb.r.Close()
}

// NewVPNChecker creates a new checker that can be asked for VPN IPs.
// it connects to the redis database for caching and requests information from all existing
// API endpoints that provode free VPN detections.
func NewVPNChecker(cfg *Config) (*VPNChecker, error) {

	apis := []VPN{}
	if !cfg.Offline {
		// share client with all apis
		httpClient := &http.Client{}

		if cfg.IPHubToken != "" {
			apis = append(apis, NewIPHub(httpClient, cfg.IPHubToken))
		}

		apis = append(apis, NewIPTeohIO(httpClient))
	}

	ripr, err := goripr.NewClient(goripr.Options{
		Addr:     cfg.RedisAddress,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
	if err != nil {
		return nil, err
	}

	return &VPNChecker{
		r:       ripr,
		Apis:    apis,
		Offline: cfg.Offline,
	}, nil
}

//
func (rdb *VPNChecker) foundInCache(sIP string) (found bool, isVPN bool, reason string, err error) {

	reason, err = rdb.r.Find(sIP)
	if errors.Is(goripr.ErrIPNotFound, err) {
		return false, false, reason, nil
	} else if err != nil {
		return false, false, "", err
	}

	return true, true, reason, nil
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

	IsVPN = percentage >= 0.75
	return
}

// IsVPN checks firstly in cache and then online.
func (rdb *VPNChecker) IsVPN(sIP string) (bool, string, error) {

	IP := net.ParseIP(sIP).To4().String()
	if IP == "<nil>" {
		return false, "", errors.New("Invalid IP passed, expexted IPv4")
	}

	found, isVPN, reason, err := rdb.foundInCache(IP)
	if err != nil {
		return false, "", err
	}

	if found {
		log.Println("[in cache]: ", IP)
		return isVPN, reason, nil
	} else {
		log.Println("[not in cache]: ", IP)
	}

	// not found, lookup online
	if rdb.Offline {
		log.Println("[skipping online check]: ", IP)
		// if the detection is offline, cache only,
		// caching of default no values makes no sense, so no caching here.
		return false, "", nil
	}

	isOnlineVPN := rdb.foundOnline(IP)
	log.Printf("[online]:  %s\n", IP)
	// update cache values
	if isOnlineVPN {
		// forever vpn
		e := rdb.r.Insert(IP, "VPN (f/o)")
		if e != nil {
			log.Println("[error]: failed to insert VPN IP found online: ", IP)
		}
	}
	// else case, not found online
	return isOnlineVPN, "", nil // reason 1 -> VPN
}
