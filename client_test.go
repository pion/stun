package stun

import (
	"bytes"
	"encoding/base64"
	"encoding/csv"
	"hash/crc64"
	"io"
	"net"
	"strconv"
	"testing"
)

type stubMultiplexer struct {
	buf      []byte
	response []byte
	addr     net.Addr
	clients  []multiplexClient
}

func (m *stubMultiplexer) Add(f MultiplexFunc, c func([]byte, net.Addr)) {
	m.clients = append(m.clients, multiplexClient{
		f: f,
		c: c,
	})
}

func (m *stubMultiplexer) WriteTo(b []byte, addr net.Addr) (int, error) {
	m.buf = append(m.buf[:0], b...)
	m.addr = addr
	go func() {
		for _, client := range m.clients {
			if !client.f(m.response) {
				continue
			}
			client.c(m.response, m.addr)
		}
	}()
	return len(b), nil
}

func BenchmarkClient_AddTransaction(b *testing.B) {
	b.ReportAllocs()
	c := &Client{}
	id := transactionID{}
	for i := 0; i < b.N; i++ {
		c.addTransaction(id)
	}
}

func BenchmarkClient_Do(b *testing.B) {
	b.ReportAllocs()
	client := &Client{}
	m := &stubMultiplexer{
		addr: &net.UDPAddr{
			IP: net.IPv4(1, 2, 3, 4),
		},
	}
	client.Multiplex(m)
	if err := client.Dial(m.addr); err != nil {
		b.Error(err)
	}

	// file contains udp-packets captured from browsers (WebRTC)
	reader := csv.NewReader(bytes.NewReader(loadData(b, "frombrowsers.csv")))
	reader.Comma = ','
	_, err := reader.Read() // skipping header
	if err != nil {
		b.Fatal("failed to skip header of csv: ", err)
	}
	crcTable := crc64.MakeTable(crc64.ISO)
	response := New()
	for {
		line, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			b.Fatal("failed to read csv line: ", err)
		}
		data, err := base64.StdEncoding.DecodeString(line[1])
		if err != nil {
			b.Fatal("failed to decode ", line[1], " as base64: ", err)
		}
		crc, err := strconv.ParseUint(line[2], 10, 64)
		if err != nil {
			b.Fatal(err)
		}
		if crc != crc64.Checksum(data, crcTable) {
			b.Error("crc64 check failed for ", line[1])
		}
		if _, err = response.Write(data); err != nil {
			b.Error("failed to decode ", line[1], " as message: ", err)
		}
		break
	}

	request, err := Build(NewTransactionIDSetter(response.TransactionID), BindingRequest)
	if err != nil {
		b.Fatal(err)
	}
	m.response = response.Raw
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := client.Do(request, func(res *Message) error {
			return nil
		}); err != nil {
			b.Fatal(err)
		}
	}
}

func TestClient_Do(t *testing.T) {
	client := &Client{}
	m := &stubMultiplexer{
		addr: &net.UDPAddr{
			IP: net.IPv4(1, 2, 3, 4),
		},
	}
	client.Multiplex(m)
	if err := client.Dial(m.addr); err != nil {
		t.Error(err)
	}
	request, err := Build(TransactionID, BindingRequest)
	if err != nil {
		t.Fatal(err)
	}
	response, err := Build(NewTransactionIDSetter(request.TransactionID), BindingSuccess)
	if err != nil {
		t.Fatal(err)
	}
	m.response = response.Raw
	if err := client.Do(request, func(res *Message) error {
		if !res.Equal(response) {
			t.Error("not equal")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}
