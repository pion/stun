package stun

import (
	"net"
	"testing"
)

func BenchmarkBasicProcess(b *testing.B) {
	m := AcquireMessage()
	res := AcquireMessage()
	req := AcquireMessage()
	defer ReleaseMessage(res)
	defer ReleaseMessage(req)
	b.ReportAllocs()
	addr, err := net.ResolveUDPAddr("udp", "213.11.231.1:12341")
	if err != nil {
		b.Fatal(err)
	}
	m.TransactionID = NewTransactionID()
	m.AddSoftware("some software")
	m.WriteHeader()
	b.SetBytes(int64(len(m.buf.B)))
	for i := 0; i < b.N; i++ {
		res.Reset()
		req.Reset()
		if err := basicProcess(addr, m.buf.B, req, res); err != nil {
			b.Fatal(err)
		}
	}
}
