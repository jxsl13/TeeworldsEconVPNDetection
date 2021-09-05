package vpn

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// NewIPHub reates a new api that can be checked for VPN IPs
func NewIPHub(c *http.Client, apikey string) *IPHub {
	return &IPHub{
		Client:  c,
		Limiter: NewRateLimiter(24*time.Hour, 1000),
		APIKey:  apikey,
	}
}

// IPHub implemets the VPNApi interface and checks whether a given IP is a vpn
type IPHub struct {
	Client  *http.Client
	Limiter *RateLimiter
	APIKey  string
}

// String implements the stringer interface
func (ih IPHub) String() string {
	return "https://iphub.info"
}

// responseData : structure of the response from this API
type iPHubResponseData struct {
	IP          string `json:"ip"`
	CountryCode string `json:"countryCode"`
	CountryName string `json:"countryName"`
	Asn         int    `json:"asn"`
	Isp         string `json:"isp"`
	Block       int    `json:"block"`
	Hostname    string `json:"hostname"`
}

// Fetch :
func (ih *IPHub) Fetch(IP string) (string, error) {

	headers := http.Header{}
	headers.Add("X-Key", ih.APIKey)

	u, _ := url.Parse("http://v2.api.iphub.info/ip/" + IP)
	request, _ := http.NewRequest("GET", u.String(), nil)
	request.Header = headers
	response, err := ih.Client.Do(request)

	if err != nil {
		return "", err
	}

	// status
	status := response.StatusCode
	if status != 200 {
		return "", fmt.Errorf("response code is not 200: %d, check your IPHub token in .env, IPHUB_TOKEN=", status)
	}

	// body
	bytes, _ := ioutil.ReadAll(response.Body)

	data := iPHubResponseData{}
	err = json.Unmarshal(bytes, &data)
	if err != nil {
		return "", err
	}

	return strconv.Itoa(data.Block), nil

}

// IsVPN tests if a given IP is a VPN IP
func (ih *IPHub) IsVPN(IP string) (bool, error) {
	if !ih.Limiter.Allow() {
		return false, errors.New("API IPHub reached the daily limit")
	}
	body, err := ih.Fetch(IP)

	if err != nil {
		return false, err
	} else if body == "1" {
		return true, nil
	}

	return false, nil
}
