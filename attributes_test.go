package stun

import "testing"

func TestMessage_AddSoftware(t *testing.T) {
	m := AcquireMessage()
	defer ReleaseMessage(m)
	v := "Client v0.0.1"
	m.AddSoftware(v)
	m.WriteHeader()

	m2 := AcquireMessage()
	defer ReleaseMessage(m2)
	if err := m2.Get(m.buf.B); err != nil {
		t.Error(err)
	}
	vRead := m.GetSoftware()
	if vRead != v {
		t.Errorf("Expected %s, got %s.", v, vRead)
	}
}

func TestMessage_GetSoftware(t *testing.T) {
	m := AcquireMessage()
	defer ReleaseMessage(m)

	v := m.GetSoftware()
	if v != "" {
		t.Errorf("%s should be blank.", v)
	}
	vByte := m.GetSoftwareBytes()
	if vByte != nil {
		t.Errorf("%s should be nil.", vByte)
	}
}
