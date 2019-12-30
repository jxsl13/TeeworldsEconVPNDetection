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

// IPTeohIO implements the VPNApi and allows to check if an ip is a vpn
type IPTeohIO struct {
	Client *http.Client
	CooldownHandler
}

// Name : Get API Name
func (it IPTeohIO) Name() string {
	return "https://ip.teoh.io"
}

// ResponseData : structure of the response from this API
type iPTheohResponseDataInt struct {
	IP           string `json:"ip"`
	Organization string `json:"organization"`
	Asn          string `json:"asn"`
	Type         string `json:"type"`
	Risk         string `json:"risk"`
	IsHosting    int    `json:"is_hosting"` // Integer Type
	VpnOrProxy   string `json:"vpn_or_proxy"`
}

type iPTheohResponseDataString struct {
	IP           string `json:"ip"`
	Organization string `json:"organization"`
	Asn          string `json:"asn"`
	Type         string `json:"type"`
	Risk         string `json:"risk"`
	IsHosting    string `json:"is_hosting"` // String Type
	VpnOrProxy   string `json:"vpn_or_proxy"`
}

// Fetch :
func (it IPTeohIO) Fetch(IP string) (string, error) {

	u, _ := url.Parse("https://ip.teoh.io/api/vpn/" + IP)
	request, _ := http.NewRequest("GET", u.String(), nil)

	response, err := it.Client.Do(request)

	if err != nil {
		debug.PrintStack()
		return "", err
	}

	// status
	status := response.StatusCode
	if status != 200 {
		it.IncreaseCooldown()
		return "", errors.New("response code is not 200: " + strconv.Itoa(status))
	}

	// body
	bytes, _ := ioutil.ReadAll(response.Body)

	data := iPTheohResponseDataInt{}
	err = json.Unmarshal(bytes, &data)

	// has different formats depending on is vpn or not vpn
	if err != nil {
		secondData := iPTheohResponseDataString{}
		err := json.Unmarshal(bytes, &secondData)
		if err != nil {
			return "", err
		}

		if secondData.IsHosting == "1" || secondData.VpnOrProxy == "yes" {
			return "yes", nil
		}
		return "no", nil
	}

	if data.IsHosting == 1 || data.VpnOrProxy == "yes" {
		return "yes", nil
	}
	return "no", nil
}

// IsVpn :
func (it IPTeohIO) IsVpn(IP string) (bool, error) {
	body, err := it.Fetch(IP)

	if err != nil {
		return false, err
	} else if body == "yes" {
		return true, nil
	}

	return false, nil
}
