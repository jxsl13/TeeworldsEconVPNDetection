package servers

import (
	"strings"
)

type HttpMasterServerList struct {
	Servers []Servers `json:"servers"`
}

func (h *HttpMasterServerList) ServerIPs() []string {
	m := make(map[string]bool, len(h.Servers)*2)
	for _, server := range h.Servers {
		for _, ip := range server.GetIPs() {
			m[ip] = true
		}
	}
	result := make([]string, 0, len(m))
	for ip := range m {
		result = append(result, ip)
	}
	return result
}

type Servers struct {
	Addresses []string `json:"addresses"`
	Location  string   `json:"location"`
	Info      Info     `json:"info"`
}

func (s *Servers) GetIPs() []string {
	m := make(map[string]bool, len(s.Addresses))
	for _, addr := range s.Addresses {
		parts := strings.SplitN(addr, "://", 2)
		if len(parts) != 2 {
			continue
		}
		parts = strings.SplitN(parts[1], ":", 2)
		if len(parts) != 2 {
			continue
		}
		m[parts[0]] = true
	}

	result := make([]string, 0, len(m))
	for addr := range m {
		result = append(result, addr)
	}
	return result
}

type Map struct {
	Name string `json:"name"`
}

type Info struct {
	MaxClients int    `json:"max_clients"`
	MaxPlayers int    `json:"max_players"`
	Passworded bool   `json:"passworded"`
	GameType   string `json:"game_type"`
	Name       string `json:"name"`
	Map        Map    `json:"map"`
}
