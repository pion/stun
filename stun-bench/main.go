package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"runtime"
	"time"

	"github.com/ernado/stun"
)

var (
	addr = flag.String("addr",
		fmt.Sprintf("127.0.0.1:%d", stun.DefaultPort),
		"addr to attack",
	)

	count int64
)

func main() {
	flag.Parse()
	runtime.GOMAXPROCS(2)
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
	m.AddRaw(stun.AttrSoftware, []byte("stun benchmark"))
	m.Encode()
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
			count++
			if count%10000 == 0 {
				fmt.Printf("%d\n", count)
				elapsed := time.Since(start)
				fmt.Println(float64(count)/elapsed.Seconds(), "per second")
			}
			// mRec.Reset()
		}
	}()
	for {
		_, err := c.Write(m.Raw)
		if err != nil {
			log.Fatalln("write:", err)
		}
	}
}
