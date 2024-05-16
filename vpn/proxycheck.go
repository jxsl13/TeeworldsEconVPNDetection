package vpn

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"time"
)

var _ VPN = (*ProxyCheck)(nil)

// NewProxyCheck reates a new api that can be checked for VPN IPs
func NewProxyCheck(c *http.Client, apikey string) *ProxyCheck {
	return &ProxyCheck{
		client:  c,
		limiter: NewRateLimiter(24*time.Hour, 1000),
		apiKey:  apikey,
	}
}

// ProxyCheck implemets the VPNApi interface and checks whether a given IP is a vpn
type ProxyCheck struct {
	client  *http.Client
	limiter *RateLimiter
	apiKey  string
}

// String implements the stringer interface
func (ih *ProxyCheck) String() string {
	return "proxycheck.io"
}

// responseData : structure of the response from this API (IP information)
type ProxyCheckInfoResponseData struct {
	Asn        string
	Provider   string
	Continent  string
	Country    string
	Isocode    string
	Region     string
	Regioncode string
	City       string
	Latitude   float32
	Longitude  float32
	Proxy      string
	Type       string
}

// Fetch :
func (ih *ProxyCheck) Fetch(IP string) (bool, error) {

	u := url.URL{
		Scheme: "https",
		Host:   "proxycheck.io",
		Path:   path.Join("/v2/", IP),
		RawQuery: url.Values{
			"vpn": []string{"1"},
			"asn": []string{"1"},
			"key": []string{ih.apiKey},
		}.Encode(),
	}

	request, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	response, err := ih.client.Do(request)
	if err != nil {
		return false, err
	}
	defer response.Body.Close()

	// status
	status := response.StatusCode
	if status/100 != 2 {
		return false, fmt.Errorf("response code is not 200: %d", status)
	}

	// body
	bytes, err := io.ReadAll(response.Body)
	if err != nil {
		return false, fmt.Errorf("error while reading response body: %w", err)
	}
	var data map[string]json.RawMessage
	err = json.Unmarshal(bytes, &data)
	if err != nil {
		return false, err
	}

	var statusApi string
	err = json.Unmarshal(data["status"], &statusApi)
	if err != nil {
		return false, err
	}

	if statusApi != "ok" {
		var messageApi string
		err = json.Unmarshal(data["message"], &messageApi)
		if err != nil {
			return false, err
		}

		return false, errors.New(messageApi)
	}

	responseApi := ProxyCheckInfoResponseData{}
	err = json.Unmarshal(data[IP], &responseApi)
	if err != nil {
		return false, err
	}

	return responseApi.Proxy == "yes", nil

}

// IsVPN tests if a given IP is a VPN IP
func (ih *ProxyCheck) IsVPN(IP string) (bool, error) {
	if !ih.limiter.Allow() {
		return false, ErrRateLimitReached
	}

	return ih.Fetch(IP)
}
