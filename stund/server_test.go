package main

import (
	"net"
	"testing"

	"github.com/ernado/stun"
)

func BenchmarkBasicProcess(b *testing.B) {
	m := stun.New()
	res := stun.New()
	req := stun.New()
	b.ReportAllocs()
	addr, err := net.ResolveUDPAddr("udp", "213.11.231.1:12341")
	if err != nil {
		b.Fatal(err)
	}
	m.TransactionID = stun.NewTransactionID()
	stun.NewSoftware("some software").AddTo(m)
	m.WriteHeader()
	b.SetBytes(int64(len(m.Raw)))
	for i := 0; i < b.N; i++ {
		res.Reset()
		req.Reset()
		if err := basicProcess(addr, m.Raw, req, res); err != nil {
			b.Fatal(err)
		}
	}
}
