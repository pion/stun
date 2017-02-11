package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/ernado/stun"
)

var (
	addr = flag.String("addr",
		fmt.Sprintf("127.0.0.1:%d", stun.DefaultPort),
		"addr to attack",
	)
	readWorkers  = flag.Int("read-workers", 1, "concurrent read workers")
	writeWorkers = flag.Int("write-workers", 1, "concurrent write workers")

	count int64
)

func main() {
	flag.Parse()
	log.SetFlags(log.Lshortfile)
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()
	a, err := net.ResolveUDPAddr("udp", *addr)
	if err != nil {
		log.Fatalln("resolve:", err)
	}
	c, err := net.DialUDP("udp", nil, a)
	if err != nil {
		log.Fatalln("dial:", err)
	}
	m := &stun.Message{
		Type: stun.MessageType{
			Class:  stun.ClassRequest,
			Method: stun.MethodBinding,
		},
		TransactionID: stun.NewTransactionID(),
	}
	m.Add(stun.AttrSoftware, []byte("stun benchmark"))
	m.Encode()
	for i := 0; i < *readWorkers; i++ {
		go func() {
			mRec := stun.New()
			mRec.Raw = make([]byte, 1024)
			start := time.Now()
			for {
				_, err := c.Read(mRec.Raw[:cap(mRec.Raw)])
				if err != nil {
					log.Fatalln("read back:", err)
				}
				// mRec.Raw = mRec.Raw[:n]
				// if err := mRec.Decode(); err != nil {
				// 	log.Fatalln("Decode:", err)
				// }
				atomic.AddInt64(&count, 1)
				if c := atomic.LoadInt64(&count); c%10000 == 0 {
					fmt.Printf("%d\n", c)
					elapsed := time.Since(start)
					fmt.Println(float64(c)/elapsed.Seconds(), "per second")
				}
				// mRec.Reset()
			}
		}()
	}
	for i := 1; i < *writeWorkers; i++ {
		go func() {
			for {
				_, err := c.Write(m.Raw)
				if err != nil {
					log.Fatalln("write:", err)
				}
			}
		}()
	}
	for {
		_, err := c.Write(m.Raw)
		if err != nil {
			log.Fatalln("write:", err)
		}
	}
}
