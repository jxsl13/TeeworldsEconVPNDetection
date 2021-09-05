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
	cfg      = config.New()
)

func init() {
	if !cfg.ProxyDetectionEnabled {
		log.Println("skipping proxy detection initialization, disabled...")
		return
	}
	if config.New().ProxyUpdateInterval < 10*time.Second {
		log.Println("disabled registered Teeworlds proxy IPs check, increase the update interval to above 10s in order to enable.")
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
	// fetch udp server list
	addresses, err := browser.GetServerAddresses()
	if err != nil {
		return err
	}

	// add master server IPs
	mu.Lock()
	for _, addr := range addresses {
		ip := addr.IP.String()
		knownIPs[ip] = true
	}
	mu.Unlock()

	// fetch http server list
	ips, err := GetHttpServerIPs()
	if err != nil {
		return err
	}

	// add http master server IPs
	mu.Lock()
	defer mu.Unlock()
	for _, ip := range ips {
		knownIPs[ip] = true
	}

	log.Printf("known potential proxy IPs: %d\n", len(knownIPs))
	return nil
}

// IsTeeworldsServer checks whether a joining IP resembles that of a known registered Teeworlds server.
func IsTeeworldsServer(ip string) bool {
	if !cfg.ProxyDetectionEnabled {
		return false
	}
	mu.Lock()
	defer mu.Unlock()
	return knownIPs[ip]
}
