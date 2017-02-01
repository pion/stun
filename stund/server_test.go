package main

import (
	"net"
	"testing"

	"github.com/ernado/stun"
)

func BenchmarkBasicProcess(b *testing.B) {
	m := stun.AcquireMessage()
	res := stun.AcquireMessage()
	req := stun.AcquireMessage()
	defer stun.ReleaseMessage(res)
	defer stun.ReleaseMessage(req)
	b.ReportAllocs()
	addr, err := net.ResolveUDPAddr("udp", "213.11.231.1:12341")
	if err != nil {
		b.Fatal(err)
	}
	m.TransactionID = stun.NewTransactionID()
	m.AddSoftware("some software")
	m.WriteHeader()
	b.SetBytes(int64(len(m.Bytes())))
	for i := 0; i < b.N; i++ {
		res.Reset()
		req.Reset()
		if err := basicProcess(addr, m.Bytes(), req, res); err != nil {
			b.Fatal(err)
		}
	}
}
