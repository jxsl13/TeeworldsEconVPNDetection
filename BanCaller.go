package main

import (
	"bytes"
	"io"
	"log"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/reiver/go-telnet"
)

type internalStandardCaller struct {
	Address   string
	Password  string
	Checker   VPNChecker
	Env       map[string]string
	WaitGroup *sync.WaitGroup
}

func (caller internalStandardCaller) CallTELNET(ctx telnet.Context, w telnet.Writer, r telnet.Reader) {
	standardCallerCallTELNET(ctx, w, r, caller.Address, caller.Password, caller.Checker, caller.Env, caller.WaitGroup)
}

// ReadLine reads until a line is read from econ
func ReadLine(reader io.Reader) (string, error) {

	// character buffer
	var buffer [1]byte
	p := buffer[:]

	// line buffer
	var underlyingBuffer []byte

	nl := []byte("\n")

	for {
		// Read 1 byte.
		n, err := reader.Read(p)
		if n <= 0 && nil == err {
			continue
		} else if n <= 0 && nil != err {
			return "", err
		}

		underlyingBuffer = append(underlyingBuffer, p...)
		// check for newline
		if bytes.Compare(p, nl) == 0 {
			line := string(underlyingBuffer)
			return line, nil
		}
	}
}

// WriteLine writes an econ command
func WriteLine(writer io.Writer, line string) {
	writer.Write([]byte(line)) // line
	writer.Write([]byte("\n")) // enter - confimation
}

// Login logs you into the econ.
func Login(reader io.Reader, writer io.Writer, password string) (bool, error) {
	line, err := ReadLine(reader)
	if err != nil {
		return false, err
	}

	if strings.Compare(line, "Enter password:\n") == 0 {
		WriteLine(writer, password)
		_, err := ReadLine(reader)
		if err != nil {
			return false, err
		}

		//log.Println(confirmation)
		return true, nil
	}
	return false, nil
}

func standardCallerCallTELNET(ctx telnet.Context, w telnet.Writer, r telnet.Reader, address string, password string, checker VPNChecker, env map[string]string, wg *sync.WaitGroup) {

	success, err := Login(r, w, password)
	if !success && err == nil {
		log.Println("Invalid Password:", password, "( Server IP:", address, ")")
		time.Sleep(1 * time.Second)
		return
	} else if err != nil {
		log.Println("Failed to log in:", err.Error(), "( Server IP:", address, ")")
		time.Sleep(1 * time.Second)
		return
	} else {
		log.Println("Successfully connected to: ", address)
	}

	regex := regexp.MustCompile(`player is ready\. ClientID=([\d]+) addr=([\d]{1,3}\.[\d]{1,3}\.[\d]{1,3}\.[\d]{1,3})`)

	for {
		line, err := ReadLine(r)

		if err != nil {
			log.Println("An error occurred while trying to read a line:", err.Error(), "(", address, ")")
			time.Sleep(5 * time.Second)
			continue
		}

		matches := regex.FindAllStringSubmatch(line, -1)
		if len(matches) > 0 {
			ID := matches[0][1]
			IP := matches[0][2]

			isVPN, err := checker.IsVPN(IP)
			if err != nil {
				log.Println(err.Error())
				continue
			}

			if isVPN {
				log.Println("[vpn] IP [is a vpn] :", IP)
				WriteLine(w, "ban "+ID+" "+env["VPN_BAN_TIME"]+" "+env["VPN_BAN_REASON"])
			} else {
				log.Println("[vpn] IP [not a vpn]:", IP)
			}
		}
	}

	log.Println("Closed connection to: ", address)
}
