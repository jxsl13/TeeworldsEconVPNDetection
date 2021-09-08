package servers

import (
	"log"
	"sort"
	"sync"
	"time"

	"github.com/jxsl13/TeeworldsEconVPNDetectionGo/config"
)

var (
	// ip -> servers
	similarServers = make(map[string][]Server, 64)
	mu             sync.Mutex
	cfg            = config.New()
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
	oldServerListSize, newServerListSize := 0, 0
	oldSimilarServers, newSimilarServers := 0, 0
	// fetch http server list
	m, err := GetSimilarServers()
	if err != nil {
		return err
	}

	// add http master server IPs
	added := 0
	mu.Lock()
	oldSimilarServers = len(similarServers)
	for ip, newServers := range m {
		oldServerListSize = len(similarServers[ip])
		// append new serers if necessary
		similarServers[ip] = appendIfNotExists(similarServers[ip], newServers...)
		newServerListSize = len(similarServers[ip])
		added += newServerListSize - oldServerListSize
	}
	newSimilarServers = len(similarServers)
	log.Printf("cached unique server IPs: %d, added %d new ips and %d new servers\n", newSimilarServers, newSimilarServers-oldSimilarServers, added)
	printSimilarServers(similarServers)
	mu.Unlock()

	return nil
}

func printSimilarServers(m map[string][]Server) {
	sortedIPs := make([]string, 0, len(m))
	for ip := range m {
		sortedIPs = append(sortedIPs, ip)
	}
	sort.Strings(sortedIPs)

	log.Println("================")
	log.Println("Similar Servers:")
	for _, ip := range sortedIPs {
		log.Printf("%15s:", ip)
		servers := m[ip]
		for _, server := range servers {
			addresses := server.Addresses
			if len(addresses) == 0 {
				log.Println("\tno addresses found...")
				break
			}
			addr := addresses[0]
			log.Printf("\t%v: %s\n", addr, server.Info.Name)
		}
	}
	log.Println("================")
}

// IsTeeworldsServer checks whether a joining IP resembles that of a known registered Teeworlds server.
func IsTeeworldsServer(ip string) bool {
	if !cfg.ProxyDetectionEnabled {
		return false
	}
	mu.Lock()
	defer mu.Unlock()
	_, found := similarServers[ip]
	return found
}
