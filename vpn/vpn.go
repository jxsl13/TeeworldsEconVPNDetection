package vpn

import (
	"errors"
	"fmt"
)

var (
	ErrRateLimitReached = errors.New("rate limit reached")
)

// VPN API interface. Provides a method to test IPs for whether they are VPNs or not.
type VPN interface {
	fmt.Stringer
	IsVPN(IP string) (bool, error)
}
