package stun

// blank is just blank string and exists just because it is ugly to keep it
// in code.
const blank = ""

// AddSoftwareBytes adds SOFTWARE attribute with value from byte slice.
func (m *Message) AddSoftwareBytes(software []byte) {
	m.Add(AttrSoftware, software)
}

// AddSoftware adds SOFTWARE attribute with value from string.
func (m *Message) AddSoftware(software string) {
	m.Add(AttrSoftware, []byte(software))
}

// GetSoftwareByte returns SOFTWARE attribute value in byte slice.
// If not found, returns nil.
func (m *Message) GetSoftwareBytes() []byte {
	return m.Attributes.Get(AttrSoftware).Value
}

// GetSoftware returns SOFTWARE attribute value in string.
// If not found, returns blank string.
func (m *Message) GetSoftware() string {
	v := m.GetSoftwareBytes()
	if len(v) == 0 {
		return blank
	}
	return string(v)
}
