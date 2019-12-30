package main

// VpnAPI :
type VpnAPI interface {
	Name() string
	IsVpn(IP string) (bool, error)
	Fetch(IP string) (string, error)
}
