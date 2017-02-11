package stun

// Software is SOFTWARE attribute.
type Software struct {
	Raw []byte
}

func (s *Software) String() string {
	return string(s.Raw)
}

// NewSoftware returns *Software from string.
func NewSoftware(software string) *Software {
	return &Software{Raw: []byte(software)}
}

// AddTo adds Software attribute to m.
func (s *Software) AddTo(m *Message) error {
	m.Add(AttrSoftware, m.Raw)
	return nil
}

// GetFrom decodes Software from m.
func (s *Software) GetFrom(m *Message) error {
	v, err := m.Get(AttrSoftware)
	if err != nil {
		return err
	}
	s.Raw = v
	return nil
}
