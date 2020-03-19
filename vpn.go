package main

// VPN API interface. Provides a method to test IPs for whether they are VPNs or not.
type VPN interface {
	String() string
	IsVPN(IP string) (bool, error)
}
