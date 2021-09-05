package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/jxsl13/TeeworldsEconVPNDetectionGo/config"
	"github.com/jxsl13/TeeworldsEconVPNDetectionGo/econ"
)

func main() {
	cfg := config.New()
	defer config.Close()

	// start goroutines
	for idx, addr := range cfg.EconServers {
		go econ.NewEvaluationRoutine(addr, cfg.EconPasswords[idx])
	}

	// block main goroutine until the application receives a signal
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM)
	<-sc
	log.Println("Shutting down...")
}
