package main

import (
	"bufio"
	"bytes"
	"io"
	"log"
	"regexp"
	"strings"

	"github.com/reiver/go-telnet"
)

// TeeworldsCaller is a simple TELNET client which sends to the server any data it gets from os.Stdin
// as TELNET (and TELNETS) data, and writes any TELNET (or TELNETS) data it receives from
// the server to os.Stdout, and writes any error it has to os.Stderr.
var TeeworldsCaller telnet.Caller = internalStandardCaller{}

type internalStandardCaller struct {
	Password string
	Checker  VPNChecker
	Env      map[string]string
}

func (caller internalStandardCaller) CallTELNET(ctx telnet.Context, w telnet.Writer, r telnet.Reader) {

	standardCallerCallTELNET(ctx, w, r, caller.Password, caller.Checker, caller.Env)
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
		confirmation, err := ReadLine(reader)
		if err != nil {
			return false, err
		}

		log.Println(confirmation)
		return true, nil
	}
	return false, nil
}

func standardCallerCallTELNET(ctx telnet.Context, w telnet.Writer, r telnet.Reader, password string, checker VPNChecker, env map[string]string) {

	success, err := Login(r, w, password)
	if !success && err == nil {
		log.Println("Invalid Password:", password)
		return
	} else if err != nil {
		log.Println("Failed to log in:", err.Error())
		return
	}

	regex := regexp.MustCompile(`player is ready\. ClientID=([\d]+) addr=([\d]{1,3}\.[\d]{1,3}\.[\d]{1,3}\.[\d]{1,3})`)

	for {
		line, _ := ReadLine(r)

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
}

func scannerSplitFunc(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF {
		return 0, nil, nil
	}

	return bufio.ScanLines(data, atEOF)
}
