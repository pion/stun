package stun

import (
	"net"
	"testing"

	"github.com/sirupsen/logrus"
)

func newServer(t *testing.T) (*net.UDPAddr, func()) {
	laddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	con, err := net.ListenUDP("udp", laddr)
	if err != nil {
		t.Fatal(err)
	}
	addr, ok := con.LocalAddr().(*net.UDPAddr)
	if !ok {
		t.Error("not UDP addr")
	}
	s := &Server{}
	logger := logrus.New()
	logger.Level = logrus.ErrorLevel
	s.Logger = logger
	go s.Serve(con)
	return addr, func() {
		if err := con.Close(); err != nil {
			t.Error(err)
		}
	}
}

func newTestRequest(addr *net.UDPAddr, m *Message) Request {
	return Request{
		Message: m,
		Target:  addr.String(),
	}
}

func TestClientServer(t *testing.T) {
	serverAddr, closer := newServer(t)
	defer closer()
	m := AcquireFields(Message{
		TransactionID: NewTransactionID(),
		Type: MessageType{
			Method: MethodBinding,
			Class:  ClassRequest,
		},
	})
	m.AddSoftware("cydev/stun client")
	m.WriteHeader()
	r := newTestRequest(serverAddr, m)
	defer ReleaseMessage(m)
	if err := DefaultClient.Do(r, func(res Response) error {
		if res.Message.GetSoftware() != "cydev/stun" {
			t.Errorf("bad software attribute: %s", res.Message.GetSoftware())
		}
		ip, _, err := res.Message.GetXORMappedAddress()
		if err != nil {
			t.Error(err)
		}
		if !ip.Equal(net.ParseIP("127.0.0.1")) {
			t.Error("bad ip", ip)
		}
		return nil
	}); err != nil {
		t.Error(err)
	}
}
