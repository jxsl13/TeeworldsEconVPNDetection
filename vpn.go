package main

type VPN interface {
	String() string
	IsVPN(IP string) (bool, error)
}
