package main

import (
	"context"
	"crypto/rand"
	"errors"
	"flag"
	"fmt"
	"log"
	mathRand "math/rand"
	"net"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"sync/atomic"
	"time"

	"github.com/pion/stun"
)

var (
	workers    = flag.Int("w", runtime.GOMAXPROCS(0), "concurrent workers")           // nolint:gochecknoglobals
	addr       = flag.String("addr", "localhost", "target address")                   // nolint:gochecknoglobals
	port       = flag.Int("port", stun.DefaultPort, "target port")                    // nolint:gochecknoglobals
	duration   = flag.Duration("d", time.Minute, "benchmark duration")                // nolint:gochecknoglobals
	network    = flag.String("net", "udp", "protocol to use (udp, tcp)")              // nolint:gochecknoglobals
	cpuProfile = flag.String("cpuprofile", "", "file output of pprof cpu profile")    // nolint:gochecknoglobals
	memProfile = flag.String("memprofile", "", "file output of pprof memory profile") // nolint:gochecknoglobals
	realRand   = flag.Bool("crypt", false, "use crypto/rand as random source")        // nolint:gochecknoglobals
)

func main() { // nolint:gocognit
	flag.Parse()
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)
	start := time.Now()
	var (
		request    int64
		requestOK  int64
		requestErr int64
	)
	if *cpuProfile != "" {
		f, createErr := os.Create(*cpuProfile)
		if createErr != nil {
			log.Fatalln("failed to create cpu profile output file:", createErr)
		}
		if pprofErr := pprof.StartCPUProfile(f); pprofErr != nil {
			log.Fatalln("failed to start pprof cpu profiling:", pprofErr)
		}
		defer func() {
			pprof.StopCPUProfile()
			if closeErr := f.Close(); closeErr != nil {
				log.Println("failed to close cpu profile output file:", closeErr)
			} else {
				fmt.Println("saved cpu profile to", *cpuProfile)
			}
		}()
	}
	if *memProfile != "" {
		f, createErr := os.Create(*memProfile)
		if createErr != nil {
			log.Panicln("failed to create memory profile output file:", createErr)
		}
		defer func() {
			if pprofErr := pprof.Lookup("heap").WriteTo(f, 1); pprofErr != nil {
				log.Fatalln("failed to write pprof memory profiling:", pprofErr)
			}
			if closeErr := f.Close(); closeErr != nil {
				log.Println("failed to close memory profile output file:", closeErr)
			} else {
				fmt.Println("saved memory profile to", *memProfile)
			}
		}()
	}
	ctx, cancel := context.WithTimeout(context.Background(), *duration)
	go func() {
		for sig := range signals {
			fmt.Println("stopping on", sig)
			cancel()
		}
	}()
	if *realRand {
		fmt.Println("using crypto/rand as random source for transaction id")
	}
	for i := 0; i < *workers; i++ {
		wConn, connErr := net.Dial(*network, fmt.Sprintf("%s:%d", *addr, *port))
		if connErr != nil {
			log.Panicln("failed to dial:", wConn)
		}
		c, clientErr := stun.NewClient(wConn)
		if clientErr != nil {
			log.Panicln("failed to create client:", clientErr)
		}
		go func() {
			req := stun.New()
			for {
				if *realRand {
					if _, err := rand.Read(req.TransactionID[:]); err != nil {
						log.Fatal("rand.Read failed:", err)
					}
				} else {
					mathRand.Read(req.TransactionID[:]) // nolint:gosec
				}
				req.Type = stun.BindingRequest
				req.WriteHeader()
				atomic.AddInt64(&request, 1)
				if doErr := c.Do(req, func(event stun.Event) {
					if event.Error != nil {
						if !errors.Is(event.Error, stun.ErrTransactionTimeOut) {
							log.Println("event.Error error:", event.Error)
						}
						atomic.AddInt64(&requestErr, 1)
						return
					}
					atomic.AddInt64(&requestOK, 1)
				}); doErr != nil {
					if !errors.Is(doErr, stun.ErrTransactionExists) {
						log.Println("Do() error:", doErr)
					}
					atomic.AddInt64(&requestErr, 1)
				}
			}
		}()
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
}
