package stun

import (
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

const (
	envExternalBlackbox = "TEST_EXTERNAL"
)

func isFlagged(env string) bool {
	switch strings.ToUpper(os.Getenv(env)) {
	case "YES", "Y", "1", "TRUE", "ПОЧЕМУ БЫ И НЕТ?", "IF YOU INSIST":
		return true
	default:
		return false
	}
}

func skipIfNotFlagged(t *testing.T, env string) {
	if !isFlagged(env) {
		t.Skipf("Test disabled by absent environment variable %s", env)
	}
}

func TestUDPClient(t *testing.T) {
	skipIfNotFlagged(t, envExternalBlackbox)

	saddr, err := net.ResolveUDPAddr("udp", "stun.l.google.com:19302")
	if err != nil {
		t.Fatal("resolve", err)
	}
	laddr, err := net.ResolveUDPAddr("udp", "0.0.0.0:0")
	if err != nil {
		t.Fatal("local resolve", err)
	}

	conn, err := net.DialUDP("udp", laddr, saddr)
	if err != nil {
		t.Fatal("dial", err)
	}
	m := AcquireFields(Message{
		Type:          MessageType{Method: MethodBinding, Class: ClassRequest},
		TransactionID: NewTransactionID(),
	})
	m.AddSoftware("cydev/stun alpha")
	m.WriteHeader()
	timeout := 100 * time.Millisecond
	for i := 0; i < 9; i++ {
		_, err := m.WriteTo(conn)
		if err != nil {
			t.Fatal(err)
		}
		if err = conn.SetReadDeadline(time.Now().Add(timeout)); err != nil {
			t.Error(err)
		}
		if timeout < 1600*time.Millisecond {
			timeout *= 2
		}
		var (
			ip   net.IP
			port int
		)
		if err == nil {
			mRec := AcquireMessage()
			if _, err = mRec.ReadFrom(conn); err != nil {
				t.Error(err)
			}
			log.Println("got message:", mRec)
			log.Println("got attributes:", mRec.Attributes)
			log.Println("got error:", err)
			if mRec.TransactionID != m.TransactionID {
				t.Error("TransactionID missmatch")
			}
			v := mRec.getAttrValue(AttrXORMappedAddress)
			log.Println(v)
			ip, port, err = mRec.GetXORMappedAddress()
			if err != nil {
				t.Error(err)
			}
			log.Println(ip, port)
			ReleaseMessage(mRec)
			break
		} else {
			if !err.(net.Error).Timeout() {
				t.Fatal(err)
			}
		}
	}
}

func TestClientSend(t *testing.T) {
	skipIfNotFlagged(t, envExternalBlackbox)
	// stun.l.google.com:19302
	ips, err := net.LookupHost("stun.l.google.com")
	if err != nil {
		t.Fatal(ErrInvalidMagicCookie)
	}
	if len(ips) == 0 {
		t.Fatal(ips)
	}
	addr := net.JoinHostPort(ips[0], strconv.Itoa(19302))
	conn, err := net.Dial("udp", addr)
	if err != nil {
		t.Fatal(err)
	}
	m := AcquireMessage()
	m.Type = MessageType{Method: MethodBinding, Class: ClassRequest}
	m.TransactionID = NewTransactionID()
	m.AddSoftware("cydev/stun alpha")
	m.WriteHeader()
	timeout := 100 * time.Millisecond
	for i := 0; i < 9; i++ {
		_, err := m.WriteTo(conn)
		if err != nil {
			t.Fatal(err)
		}
		if err = conn.SetReadDeadline(time.Now().Add(timeout)); err != nil {
			t.Error(err)
		}
		if timeout < 1600*time.Millisecond {
			timeout *= 2
		}
		var (
			ip   net.IP
			port int
		)
		if err == nil {
			mRec := AcquireMessage()
			if _, err = mRec.ReadFrom(conn); err != nil {
				t.Error(err)
			}
			log.Println("got message:", mRec)
			log.Println("got attributes:", mRec.Attributes)
			log.Println("got error:", err)
			if mRec.TransactionID != m.TransactionID {
				t.Error("TransactionID missmatch")
			}
			v := mRec.getAttrValue(AttrXORMappedAddress)
			log.Println(v)
			ip, port, err = mRec.GetXORMappedAddress()
			if err != nil {
				t.Error(err)
			}
			log.Println(ip, port)
			ReleaseMessage(mRec)
			break
		} else {
			if !err.(net.Error).Timeout() {
				t.Fatal(err)
			}
		}
	}
}
