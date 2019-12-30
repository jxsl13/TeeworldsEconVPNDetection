package main

import (
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"runtime/debug"
	"strconv"
)

// GetIPIntelNet :
type GetIPIntelNet struct {
	Client *http.Client
	CooldownHandler
	Email     string
	Threshold float64
}

// Name : Get API Name
func (giin GetIPIntelNet) Name() string {
	return "https://getipintel.net"
}

// Fetch :
func (giin GetIPIntelNet) Fetch(IP string) (string, error) {
	params := url.Values{}
	params.Add("ip", IP)
	params.Add("contact", giin.Email)

	u, _ := url.Parse("http://check.getipintel.net/check.php")
	u.RawQuery = params.Encode()

	request, _ := http.NewRequest("GET", u.String(), nil)
	response, err := giin.Client.Do(request)

	if err != nil {
		debug.PrintStack()
		return "", err
	}

	// status
	status := response.StatusCode

	// body
	bytes, _ := ioutil.ReadAll(response.Body)
	bodyText := string(bytes)

	if status == 200 {
		giin.ResetCooldown()
		return bodyText, nil
	} else if status == 400 {

		errorCode, _ := strconv.Atoi(bodyText)

		switch errorCode {
		case -1:
			log.Println("GetIPIntelNet:", errorCode, "Invalid no input")
		case -2:
			log.Println("GetIPIntelNet:", errorCode, "Invalid IP address")
		case -3:
			log.Println("GetIPIntelNet:", errorCode, "Unroutable address / private address")
		case -4:
			log.Println("GetIPIntelNet:", errorCode, "Unable to reach database, most likely the database is being updated. Keep an eye on twitter for more information.")
		case -5:
			log.Println("GetIPIntelNet:", errorCode, "Your connecting IP has been banned from the system or you do not have permission to access a particular service. Did you exceed your query limits? Did you use an invalid email address? If you want more information, please use the contact links below.")
		case -6:
			log.Println("GetIPIntelNet:", errorCode, "You did not provide any contact information with your query or the contact information is invalid.")
		default:
			log.Println("GetIPIntelNet:", errorCode, "Unknown error code:", errorCode)
		}

	} else if status == 429 {
		errorCode, _ := strconv.Atoi(bodyText)
		log.Print("GetIPIntelNet:", errorCode, "If you exceed the number of allowed queries, you'll receive a HTTP 429 error.")
	} else {
		log.Println("GetIPIntelNet: Unknown response status code:", status)
	}
	giin.IncreaseCooldown()
	fetchErr := errors.New("Failed fetching from GetIPIntelNet")
	return "", fetchErr
}

// IsVpn :
func (giin GetIPIntelNet) IsVpn(IP string) (bool, error) {
	body, err := giin.Fetch(IP)
	if err != nil {
		log.Println(err.Error())
		return false, errors.New("failed to fetch data")
	}

	vpnProbability, err := strconv.ParseFloat(body, 64)

	if err != nil {
		log.Println("Could not convert '", body, "' to float64")
		return false, errors.New("Failed to convert retrieved value to float64")
	}

	if 0.0 <= vpnProbability && vpnProbability <= 1.0 && vpnProbability >= giin.Threshold {
		return true, nil
	}
	return false, nil
}
