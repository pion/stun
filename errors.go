package stun

import (
	"fmt"
)

// Error is error type for constant errors in stun package.
//
// See http://dave.cheney.net/2016/04/07/constant-errors for more info.
type Error string

func (e Error) Error() string {
	return string(e)
}

// DecodeErr records an error and place when it is occurred.
type DecodeErr struct {
	Place   DecodeErrPlace
	Message string
}

// IsPlaceParent reports if error place parent is p.
func (e DecodeErr) IsPlaceParent(p string) bool {
	return e.Place.Parent == p
}

// IsPlaceChildren reports if error place children is c.
func (e DecodeErr) IsPlaceChildren(c string) bool {
	return e.Place.Children == c
}

// IsPlace reports if error place is p.
func (e DecodeErr) IsPlace(p DecodeErrPlace) bool {
	return e.Place == p
}

// DecodeErrPlace records a place where error is occurred.
type DecodeErrPlace struct {
	Parent   string
	Children string
}

func (p DecodeErrPlace) String() string {
	return fmt.Sprintf("%s/%s", p.Parent, p.Children)
}

func (e DecodeErr) Error() string {
	return fmt.Sprintf("BadFormat for %s: %s",
		e.Place,
		e.Message,
	)
}

func newDecodeErr(parent, children, message string) DecodeErr {
	return DecodeErr{
		Place:   DecodeErrPlace{Parent: parent, Children: children},
		Message: message,
	}
}
