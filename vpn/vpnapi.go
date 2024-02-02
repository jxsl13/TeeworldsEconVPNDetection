package vpn

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"golang.org/x/time/rate"
)

var _ VPN = (*VPNAPI)(nil)

// NewVPNAPI creates a new api endpoint that can check IPs for whether they are VPNs or not.
func NewVPNAPI(c *http.Client, apiKey string) *VPNAPI {
	return &VPNAPI{
		client: c,
		apiKey: apiKey,
		// limiter: NewRateLimiter(24*time.Hour, 1000),
		rate: rate.NewLimiter(rate.Every(24*time.Hour), 1000),
	}
}

// VPNAPI implements the VPNApi and allows to check if an ip is a vpn
type VPNAPI struct {
	client *http.Client
	apiKey string
	// limiter *RateLimiter
	rate *rate.Limiter
}

// String implements the stinger interface
func (it VPNAPI) String() string {
	return "vpnapi.io"
}

type vpnAPIResponse struct {
	Security Security `json:"security"`
}

type Security struct {
	VPN   bool `json:"vpn"`
	Proxy bool `json:"proxy"`
	Tor   bool `json:"tor"`
	Relay bool `json:"relay"`
}

// Fetch :
func (it *VPNAPI) Fetch(IP string) (bool, error) {
	u := url.URL{
		Scheme: "https",
		Host:   "vpnapi.io",
		Path:   "/api/" + IP,
		RawQuery: url.Values{
			"key": []string{it.apiKey},
		}.Encode(),
	}

	request, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return false, err
	}

	response, err := it.client.Do(request)
	if err != nil {
		return false, err
	}
	defer response.Body.Close()

	// status
	status := response.StatusCode
	if status != 200 {
		return false, errors.New("response code is not 200: " + strconv.Itoa(status))
	}

	// body
	bytes, err := io.ReadAll(response.Body)
	if err != nil {
		return false, fmt.Errorf("error while reading response body: %w", err)
	}

	data := vpnAPIResponse{}
	err = json.Unmarshal(bytes, &data)
	if err != nil {
		return false, err
	}

	return data.Security.VPN ||
		data.Security.Proxy ||
		data.Security.Tor ||
		data.Security.Relay, nil
}

// IsVPN requests the api endpoint to test whether an IP is a VPN
func (it *VPNAPI) IsVPN(IP string) (bool, error) {
	if !it.rate.Allow() {
		return false, ErrRateLimitReached
	}

	return it.Fetch(IP)
}
