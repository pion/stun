// SPDX-FileCopyrightText: 2023 The Pion community <https://pion.ly>
// SPDX-License-Identifier: MIT

// Package main implements benchmarks for the STUN package
package main

import (
	"context"
	"crypto/rand"
	"errors"
	"flag"
	"log"
	mathRand "math/rand"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"sync/atomic"
	"time"

	"github.com/pion/stun/v3"
)

var (
	workers    = flag.Int("w", runtime.GOMAXPROCS(0), "concurrent workers")           //nolint:gochecknoglobals
	uriStr     = flag.String("uri", "stun:localhost:3478", "URI of STUN server")      //nolint:gochecknoglobals
	duration   = flag.Duration("d", time.Minute, "benchmark duration")                //nolint:gochecknoglobals
	cpuProfile = flag.String("cpuprofile", "", "file output of pprof cpu profile")    //nolint:gochecknoglobals
	memProfile = flag.String("memprofile", "", "file output of pprof memory profile") //nolint:gochecknoglobals
	realRand   = flag.Bool("crypt", false, "use crypto/rand as random source")        //nolint:gochecknoglobals
)

func main() { //nolint:gocognit
	flag.Parse()
	uri, err := stun.ParseURI(*uriStr)
	if err != nil {
		log.Fatalf("Failed to parse URI '%s': %s", *uriStr, err)
	}
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
			log.Fatalf("Failed to create CPU profile output file: %s", createErr)
		}
		if pprofErr := pprof.StartCPUProfile(f); pprofErr != nil {
			log.Fatalf("Failed to start pprof CPU profiling: %s", pprofErr)
		}
		defer func() {
			pprof.StopCPUProfile()
			if closeErr := f.Close(); closeErr != nil {
				log.Printf("Failed to close CPU profile output file: %s", closeErr)
			} else {
				log.Printf("Saved cpu profile to: %s", *cpuProfile)
			}
		}()
	}
	if *memProfile != "" {
		f, createErr := os.Create(*memProfile)
		if createErr != nil {
			log.Panicf("Failed to create memory profile output file: %s", createErr)
		}
		defer func() {
			if pprofErr := pprof.Lookup("heap").WriteTo(f, 1); pprofErr != nil {
				log.Fatalf("Failed to write pprof memory profiling: %s", pprofErr)
			}
			if closeErr := f.Close(); closeErr != nil {
				log.Printf("Failed to close memory profile output file: %s", closeErr)
			} else {
				log.Printf("Saved memory profile to %s", *memProfile)
			}
		}()
	}
	ctx, cancel := context.WithTimeout(context.Background(), *duration)
	go func() {
		for sig := range signals {
			log.Printf("Stopping on %s", sig)
			cancel()
		}
	}()
	if *realRand {
		log.Print("Using crypto/rand as random source for transaction id")
	}
	for i := 0; i < *workers; i++ {
		c, clientErr := stun.DialURI(uri, &stun.DialConfig{})
		if clientErr != nil {
			log.Panicf("Failed to create client: %s", clientErr)
		}
		go func() {
			req := stun.New()
			for {
				if *realRand {
					if _, err := rand.Read(req.TransactionID[:]); err != nil { //nolint:gosec
						log.Fatalf("Failed to generate transaction ID: %s", err)
					}
				} else {
					mathRand.Read(req.TransactionID[:]) //nolint:gosec
				}
				req.Type = stun.BindingRequest
				req.WriteHeader()
				atomic.AddInt64(&request, 1)
				if doErr := c.Do(req, func(event stun.Event) {
					if event.Error != nil {
						if !errors.Is(event.Error, stun.ErrTransactionTimeOut) {
							log.Printf("Failed STUN transaction: %s", event.Error)
						}
						atomic.AddInt64(&requestErr, 1)
						return
					}
					atomic.AddInt64(&requestOK, 1)
				}); doErr != nil {
					if !errors.Is(doErr, stun.ErrTransactionExists) {
						log.Printf("Failed STUN transaction: %s", doErr)
					}
					atomic.AddInt64(&requestErr, 1)
				}
			}
		}()
	}
	log.Print("Workers started")
	<-ctx.Done()
	stop := time.Now()
	rps := int(float64(atomic.LoadInt64(&requestOK)) / stop.Sub(start).Seconds())
	log.Printf("RPS: %v", rps)
	if reqErr := atomic.LoadInt64(&requestErr); requestErr != 0 {
		log.Printf("Errors: %d", reqErr)
	}
	log.Printf("Total: %d", atomic.LoadInt64(&request))
}
