package vpn

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

var _ VPN = (*IPHub)(nil)

// NewIPHub reates a new api that can be checked for VPN IPs
func NewIPHub(c *http.Client, apikey string) *IPHub {
	return &IPHub{
		client:  c,
		limiter: NewRateLimiter(24*time.Hour, 1000),
		headers: http.Header{
			"X-Key": []string{apikey},
		},
	}
}

// IPHub implemets the VPNApi interface and checks whether a given IP is a vpn
type IPHub struct {
	client  *http.Client
	limiter *RateLimiter
	headers http.Header
}

// String implements the stringer interface
func (*IPHub) String() string {
	return "iphub.info"
}

// responseData : structure of the response from this API
type iPHubResponseData struct {
	IP          string `json:"ip"`
	CountryCode string `json:"countryCode"`
	CountryName string `json:"countryName"`
	Asn         int    `json:"asn"`
	Isp         string `json:"isp"`
	Block       int    `json:"block"`
	// Hostname    string `json:"hostname"` // deprecated
}

// Fetch :
func (ih *IPHub) Fetch(IP string) (block int, err error) {

	// for https we need to reuse an existing https connection in order not to
	// stress the api endpoint with too many tls handshakes
	// default client reuses tls connections
	u := url.URL{
		Scheme: "https",
		Host:   "v2.api.iphub.info",
		Path:   "/ip/" + IP,
	}

	req, err := http.NewRequest(http.MethodGet, u.String(), http.NoBody)
	if err != nil {
		return 0, fmt.Errorf("error while creating request: %w", err)
	}
	req.Header = ih.headers
	response, err := ih.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("error while fetching IPHub: %w", err)
	}
	defer response.Body.Close()

	// status
	status := response.StatusCode
	if status/100 != 2 {
		return 0, fmt.Errorf("response code is not 200: %d, check your IPHub token in .env, IPHUB_TOKEN=", status)
	}

	// body
	bytes, err := io.ReadAll(response.Body)
	if err != nil {
		return 0, fmt.Errorf("error while reading response body: %w", err)
	}

	data := iPHubResponseData{}
	err = json.Unmarshal(bytes, &data)
	if err != nil {
		return 0, err
	}

	return data.Block, nil

}

// IsVPN tests if a given IP is a VPN IP
func (ih *IPHub) IsVPN(IP string) (bool, error) {
	if !ih.limiter.Allow() {
		return false, ErrRateLimitReached
	}

	// https://iphub.info/api
	block, err := ih.Fetch(IP)

	if err != nil {
		return false, err
	}

	return block == 1, nil
}
