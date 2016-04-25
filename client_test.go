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
	mType := MessageType{Method: MethodBinding, Class: ClassRequest}
	m := Message{
		Type:          mType,
		Length:        0,
		TransactionID: NewTransactionID(),
	}
	buf := make([]byte, 256)
	m.Put(buf)
	if _, err := conn.Write(buf); err != nil {
		t.Fatal(err)
	}
	timeout := 100
	for i := 0; i < 9; i++ {
		_, err := conn.Write(buf)
		if err != nil {
			t.Fatal(err)
		}
		conn.SetReadDeadline(time.Now().Add(time.Duration(timeout) * time.Millisecond))
		if timeout < 1600 {
			timeout *= 2
		}
		_, err = conn.Read(buf)
		if err == nil {
			kek := Message{}
			if err = kek.Get(buf); err != nil {
				t.Error(err)
			}
			log.Println(kek)
			log.Println(kek.Attributes)
			if kek.TransactionID != m.TransactionID {
				t.Error("TransactionID missmatch")
			}
			break
		} else {
			if !err.(net.Error).Timeout() {
				t.Fatal(err)
			}
		}
	}
}
