package servers

import (
	"log"
	"sync"
	"time"

	"github.com/jxsl13/TeeworldsEconVPNDetectionGo/config"
	"github.com/jxsl13/twapi/browser"
)

var (
	knownIPs = make(map[string]bool, 1024)
	mu       sync.Mutex
)

func init() {
	if config.New().ProxyUpdateInterval < time.Minute {
		log.Println("disabled registered Teeworlds proxy IPs check, increase the update interval to above 1m in order to enable.")
		return
	}
	err := Update()
	if err != nil {
		log.Printf("failed to initialize registered Teeworlds server IP list, scheduler will not be started: %v\n", err)
		return
	}
	go func() {
		cfg := config.New()
		ticker := cfg.UpdateIPsTicker()
		for {
			select {
			case <-cfg.Context().Done():
				return
			case <-ticker.C:
				err := Update()
				if err != nil {
					log.Printf("error: failed to update registered Teeworlds server IP list: %v\n", err)
				}
			}
		}
	}()
}

// Update updates the teeworlds server list and fetches all of those IPs
func Update() error {
	addresses, err := browser.GetServerAddresses()
	if err != nil {
		return err
	}
	mu.Lock()
	defer mu.Unlock()
	for _, addr := range addresses {
		knownIPs[addr.IP.String()] = true
	}
	log.Printf("known potential proxy IPs: %d\n", len(knownIPs))
	return nil
}

// IsTeeworldsServer checks whether a joining IP resembles that of a known registered Teeworlds server.
func IsTeeworldsServer(ip string) bool {
	mu.Lock()
	defer mu.Unlock()
	return knownIPs[ip]
}
