package vpn

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"
)

// NewProxyCheck reates a new api that can be checked for VPN IPs
func NewProxyCheck(c *http.Client, apikey string) *ProxyCheck {
	return &ProxyCheck{
		Client:  c,
		Limiter: NewRateLimiter(24*time.Hour, 1000),
		APIKey:  apikey,
	}
}

// ProxyCheck implemets the VPNApi interface and checks whether a given IP is a vpn
type ProxyCheck struct {
	Client  *http.Client
	Limiter *RateLimiter
	APIKey  string
}

// String implements the stringer interface
func (ih ProxyCheck) String() string {
	return "https://proxycheck.io"
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
func (ih *ProxyCheck) Fetch(IP string) (string, error) {

	u, _ := url.Parse("https://proxycheck.io/v2/" + IP + "?vpn=1&asn=1&key=" + ih.APIKey)
	request, _ := http.NewRequest("GET", u.String(), nil)
	response, err := ih.Client.Do(request)

	if err != nil {
		return "", err
	}

	// status
	status := response.StatusCode
	if status != 200 {
		return "", fmt.Errorf("response code is not 200: %d, check your ProxyCheck token in .env, PROXYCHECK_TOKEN=", status)
	}

	// body
	bytes, _ := ioutil.ReadAll(response.Body)

	var data map[string]json.RawMessage
	err = json.Unmarshal(bytes, &data)
	if err != nil {
		return "", err
	}

	var statusApi string
	err = json.Unmarshal(data["status"], &statusApi)
	if err != nil {
		return "", err
	}

	if statusApi != "ok" {
		var messageApi string
		err = json.Unmarshal(data["message"], &messageApi)
		if err != nil {
			return "", err
		}

		return "", errors.New(messageApi)
	}

	responseApi := ProxyCheckInfoResponseData{}
	err = json.Unmarshal(data[IP], &responseApi)
	if err != nil {
		return "", err
	}

	return responseApi.Proxy, nil

}

// IsVPN tests if a given IP is a VPN IP
func (ih *ProxyCheck) IsVPN(IP string) (bool, error) {
	if !ih.Limiter.Allow() {
		return false, errors.New("API ProxyCheck reached the daily limit")
	}
	body, err := ih.Fetch(IP)

	if err != nil {
		return false, err
	} else if body == "yes" {
		return true, nil
	}

	return false, nil
}
