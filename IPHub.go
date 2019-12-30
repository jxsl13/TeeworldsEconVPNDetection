package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"runtime/debug"
	"strconv"
)

// IPHub implemets the VPNApi interface and checks whether a given IP is a vpn
type IPHub struct {
	Client  *http.Client
	Limiter *RateLimiter
	APIKey  string
}

// Name : Get API Name
func (ih IPHub) Name() string {
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
		debug.PrintStack()
		return "", err
	}

	// status
	status := response.StatusCode
	if status != 200 {
		return "", errors.New("response code is not 200: " + strconv.Itoa(status) + " check your IPHub token in .env, IPHUB_TOKEN=...")
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

// IsVpn :
func (ih *IPHub) IsVpn(IP string) (bool, error) {
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
