package main

import (
	"os"

	"github.com/jxsl13/TeeworldsEconVPNDetection/cmd"
)

func main() {
	err := cmd.NewRootCmd().Execute()
	if err != nil {
		os.Exit(1)
	}
}
