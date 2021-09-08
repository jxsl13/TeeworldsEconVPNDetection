package servers

import (
	"strings"

	"github.com/agnivade/levenshtein"
	"github.com/jxsl13/TeeworldsEconVPNDetectionGo/config"
	"golang.org/x/text/unicode/norm"
)

type HttpMasterServerList struct {
	Servers []Server `json:"servers"`
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

// IP -> ServerInfo (ip:port)
func (h *HttpMasterServerList) HostServers() map[string][]Server {
	m := make(map[string][]Server, len(h.Servers))
	for _, server := range h.Servers {
		for _, ip := range server.GetIPs() {
			_, found := m[ip]
			if !found {
				m[ip] = make([]Server, 0, 4)
			}
			m[ip] = append(m[ip], server)
		}
	}
	return m
}

// IP -> ServerInfo (ip:port), split into my servers and other servers that are not monitored via this
// tool
func (h *HttpMasterServerList) SplitHostServers() (my, other map[string][]Server) {
	allHosts := h.HostServers()
	myIPsMap := config.New().EconServerIPs()

	my = make(map[string][]Server)
	other = make(map[string][]Server, len(allHosts))

	for ip, servers := range allHosts {
		if myIPsMap[ip] {
			_, found := my[ip]
			if !found {
				my[ip] = make([]Server, 0, len(servers))
			}
			my[ip] = append(my[ip], servers...)
		} else {
			_, found := other[ip]
			if !found {
				other[ip] = make([]Server, 0, len(servers))
			}
			other[ip] = append(other[ip], servers...)
		}
	}
	return my, other
}

func (h *HttpMasterServerList) SimilarServers() (similarServers map[string][]Server) {
	my, other := h.SplitHostServers()
	similarServers = make(map[string][]Server, 16)
	upperBoundDistance := config.New().ProxyServerNameDistance

	for _, myServers := range my {
		for _, myServer := range myServers {
			for otherIP, otherServers := range other {
				for _, otherServer := range otherServers {
					distance := Distance(myServer.Info.Name, otherServer.Info.Name)
					if distance <= upperBoundDistance {
						_, found := similarServers[otherIP]
						if !found {
							similarServers[otherIP] = make([]Server, 0, 1)
						}
						similarServers[otherIP] = appendIfNotExists(similarServers[otherIP], otherServer)
					}
				}
			}
		}
	}

	return similarServers
}

func Distance(a, b string) int {
	// https://dzone.com/articles/proper-strings-normalization-for-comparison-purpos
	// https://go.dev/blog/normalization
	trimmedA := strings.TrimSpace(a)
	trimmedB := strings.TrimSpace(b)

	lcA := strings.ToLower(trimmedA)
	lcB := strings.ToLower(trimmedB)

	diacriticsA := norm.NFD.String(lcA)
	diacriticsB := norm.NFD.String(lcB)

	ligaturesA := norm.NFKD.String(diacriticsA)
	ligaturesB := norm.NFKD.String(diacriticsB)

	punctuationA := strings.ReplaceAll(ligaturesA, "—", "-")
	punctuationB := strings.ReplaceAll(ligaturesB, "—", "-")

	return levenshtein.ComputeDistance(punctuationA, punctuationB)
}

func appendIfNotExists(olds []Server, others ...Server) []Server {
	result := make([]Server, 0, len(olds)+len(others))
	result = append(result, olds...)
	found := false

	for _, other := range others {
		found = false
		for _, old := range olds {
			if old.Equals(other) {
				found = true
				break
			}
		}
		if found {
			continue
		}
		result = append(result, other)
	}

	return result
}

type Server struct {
	Addresses []string `json:"addresses"`
	Location  string   `json:"location"`
	Info      Info     `json:"info"`
}

func (s *Server) Equals(other Server) bool {
	if len(s.Addresses) != len(other.Addresses) {
		return false
	}
	for idx, addr := range s.Addresses {
		if addr != other.Addresses[idx] {
			return false
		}
	}
	return s.Location == other.Location && s.Info.Equals(other.Info)
}

func (s *Server) GetIPs() []string {
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

type Info struct {
	MaxClients int    `json:"max_clients"`
	MaxPlayers int    `json:"max_players"`
	Passworded bool   `json:"passworded"`
	GameType   string `json:"game_type"`
	Name       string `json:"name"`
	Map        Map    `json:"map"`
}

func (i *Info) Equals(other Info) bool {
	return i.MaxClients == other.MaxClients &&
		i.MaxPlayers == other.MaxPlayers &&
		i.Passworded == other.Passworded &&
		i.GameType == other.GameType &&
		i.Name == other.Name &&
		i.Map.Equals(other.Map)
}

type Map struct {
	Name string `json:"name"`
}

func (m *Map) Equals(other Map) bool {
	return m.Name == other.Name
}
