package vpn

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/netip"

	"github.com/jxsl13/goripr/v2"
)

// Valid is used to represent the answer of an api endpoint
type Valid struct {
	IsValid bool
	IsVPN   bool
}

// VPNChecker encapsulates the redis database as cache and the
// implemented api endpoints in order to determine, whether an ip is a vpn based on
// either the caching information or based on the implemented api endpoints, whether
// an ip is a vpn.
type VPNChecker struct {
	ctx context.Context
	r   *goripr.Client

	apis      []VPN
	offline   bool
	threshold float64

	wl *Whitelister
}

func (rdb *VPNChecker) Close() error {
	return rdb.r.Close()
}

// newVPNChecker creates a new checker that can be asked for VPN IPs.
// it connects to the redis database for caching and requests information from all existing
// API endpoints that provode free VPN detections.
func NewVPNChecker(
	ctx context.Context,
	ripr *goripr.Client,
	wl *Whitelister,
	vpns []VPN,
	offline bool,
	permabanThreshold float64,
) *VPNChecker {
	return &VPNChecker{
		ctx:       ctx,
		r:         ripr,
		apis:      vpns,
		offline:   offline,
		threshold: permabanThreshold,
		wl:        wl,
	}
}

func (rdb *VPNChecker) foundInCache(sIP string) (found bool, isVPN bool, reason string, err error) {

	reason, err = rdb.r.Find(rdb.ctx, sIP)
	if errors.Is(err, goripr.ErrIPNotFound) {
		return false, false, reason, nil
	} else if err != nil {
		return false, false, "", err
	}

	return true, true, reason, nil
}

func (rdb *VPNChecker) foundOnline(sIP string) (IsVPN bool) {

	results := make([]Valid, len(rdb.apis))

	for idx, api := range rdb.apis {

		isVPNTmp, err := api.IsVPN(sIP)
		if err != nil {
			log.Println("[ERROR]:", api.String(), ":", err.Error())
			results[idx] = Valid{
				IsValid: false,
				IsVPN:   false,
			}
			continue
		}

		results[idx] = Valid{
			IsValid: true,
			IsVPN:   isVPNTmp,
		}
	}

	total := 0.0
	trueValue := 0.0
	for _, valid := range results {
		if valid.IsValid {
			total += 1.0
			if valid.IsVPN {
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

	return percentage >= float64(rdb.threshold)
}

// IsVPN checks firstly in cache and then online.
func (rdb *VPNChecker) IsVPN(sIP string) (bool, string, error) {

	ip, err := netip.ParseAddr(sIP)
	if err != nil {
		return false, "", fmt.Errorf("invalid IP passed: %s: %w", sIP, err)
	}
	if !ip.Is4() {
		return false, "", fmt.Errorf("invalid IP passed, expected IPv4, got: %s", sIP)
	}

	IPStr := ip.String()

	found, isVPN, reason, err := rdb.foundInCache(IPStr)
	if err != nil {
		return false, "", err
	}

	if found {
		log.Println("[in cache]: ", IPStr)
		return isVPN, reason, nil
	}

	log.Println("[not in cache]: ", IPStr)

	// not found, lookup online
	if rdb.offline {
		log.Println("[skipping online check]:", IPStr)
		// if the detection is offline, cache only,
		// caching of default no values makes no sense, so no caching here.
		return false, "", nil
	}

	// found nuts?
	found, err = rdb.wl.Exists(IPStr)
	if err != nil {
		log.Printf("[error]: %v", err)
		return false, "", err
	}

	if found {
		log.Println("[whitelisted]: ", IPStr)
		return false, "", nil
	}
	log.Println("[not whitelisted]: ", IPStr)

	isOnlineVPN := rdb.foundOnline(IPStr)
	log.Printf("[online]:  %s\n", IPStr)
	// update cache values
	if isOnlineVPN {
		// forever vpn
		e := rdb.r.Insert(rdb.ctx, IPStr, "VPN (f/o)")
		if e != nil {
			log.Printf("[error]: failed to insert VPN IP found online: %s: %v", IPStr, e)
		}
	} else {
		// not vpn, cache in whitelist
		e := rdb.wl.Whitelist(IPStr)
		if e != nil {
			log.Printf("[error]: %v", e)
		}
	}

	// else case, not found online
	return isOnlineVPN, "", nil // reason 1 -> VPN
}
