package stun

// Setter sets *Message attribute.
type Setter interface {
	AddTo(m *Message) error
}

// Getter decodes *Message attribute.
type Getter interface {
	GetFrom(m *Message) error
}

// Checker checks *Message attribute.
type Checker interface {
	Check(m *Message) error
}

// Build applies setters to message.
func (m *Message) Build(setters... Setter) error {
	m.Reset()
	m.WriteHeader()
	for _, s := range setters {
		if err := s.AddTo(m); err != nil {
			return err
		}
	}
	return nil
}

func (m *Message) Check(checkers... Checker) error {
	for _, c := range checkers {
		if err := c.Check(m); err != nil {
			return err
		}
	}
	return nil
}

// Build wraps Message.Build method.
func Build(setters... Setter) (*Message, error) {
	m := new(Message)
	return m, m.Build(setters...)
}
