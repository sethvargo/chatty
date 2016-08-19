package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
)

var (
	listenPtr    = flag.String("listen", ":8080", "address and port to listen")
	redisAddrPtr = flag.String("redis", ":6379", "address and port to redis")
)

func main() {
	flag.Parse()

	server, err := NewServer(&NewServerOpts{
		Listen: *listenPtr,
		Redis:  *redisAddrPtr,
	})

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
			os.Exit(1)
		case s := <-signalCh:
			switch s {
			case syscall.SIGINT:
				server.Stop()
				os.Exit(2)
			}
		}
	}
}
