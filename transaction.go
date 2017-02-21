package stun

type transactionIDSetter bool

func (transactionIDSetter) AddTo(m *Message) error {
	return m.NewTransactionID()
}

// TransactionID is Setter for m.TransactionID.
var TransactionID Setter = transactionIDSetter(true)
