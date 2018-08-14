package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/gortc/stun"
)

var (
	workers  = flag.Int("w", runtime.GOMAXPROCS(0), "concurrent workers")
	addr     = flag.String("addr", fmt.Sprintf("localhost"), "target address")
	port     = flag.Int("port", stun.DefaultPort, "target port")
	duration = flag.Duration("d", time.Minute, "benchmark duration")
	network  = flag.String("net", "udp", "protocol to use (udp, tcp)")
)

func main() {
	flag.Parse()
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)
	start := time.Now()
	var (
		request    int64
		requestOK  int64
		requestErr int64
	)
	ctx, cancel := context.WithTimeout(context.Background(), *duration)
	go func() {
		for sig := range signals {
			fmt.Println("stopping on", sig)
			cancel()
		}
	}()
	for i := 0; i < *workers; i++ {
		wConn, connErr := net.Dial(*network, fmt.Sprintf("%s:%d", *addr, *port))
		if connErr != nil {
			log.Fatalln("failed to dial:", wConn)
		}
		c, clientErr := stun.NewClient(wConn)
		if clientErr != nil {
			log.Fatalln("failed to create client:", clientErr)
		}
		go func(client *stun.Client) {
			req := stun.New()
			for {
				rand.Read(req.TransactionID[:])
				req.Type = stun.BindingRequest
				req.WriteHeader()
				atomic.AddInt64(&request, 1)
				if doErr := c.Do(req, func(event stun.Event) {
					if event.Error != nil {
						log.Println("event.Error error:", event.Error)
						atomic.AddInt64(&requestErr, 1)
						return
					}
					atomic.AddInt64(&requestOK, 1)
				}); doErr != nil {
					log.Println("Do() error:", doErr)
					atomic.AddInt64(&requestErr, 1)
				}
			}
		}(c)
	}
	fmt.Println("workers started")
	<-ctx.Done()
	stop := time.Now()
	rps := int(float64(atomic.LoadInt64(&requestOK)) / stop.Sub(start).Seconds())
	fmt.Println("rps:", rps)
	if reqErr := atomic.LoadInt64(&requestErr); requestErr != 0 {
		fmt.Println("errors:", reqErr)
	}
	fmt.Println("total:", atomic.LoadInt64(&request))
	os.Exit(1)
}
