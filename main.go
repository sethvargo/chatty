package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	server, err := NewServer()
	if err != nil {
		log.Printf("[ERR] error starting server: %s", err)
		os.Exit(127)
	}

	signalCh := make(chan os.Signal, syscall.SIGINT)
	signal.Notify(signalCh)

	errCh := make(chan error)
	go func() {
		if err := server.Start(); err != nil {
			errCh <- err
		}
	}()

	for {
		select {
		case err := <-errCh:
			log.Printf("[ERR] %s", err)
		case s := <-signalCh:
			switch s {
			case syscall.SIGINT:
				server.Stop()
				os.Exit(2)
			}
		}
	}
}
