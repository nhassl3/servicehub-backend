package validator

import (
	"buf.build/go/protovalidate"
	"google.golang.org/protobuf/proto"
)

// Validator wraps protovalidate for use in interceptors.
type Validator struct {
	v protovalidate.Validator
}

// New creates a new Validator instance.
func New() (*Validator, error) {
	v, err := protovalidate.New()
	if err != nil {
		return nil, err
	}
	return &Validator{v: v}, nil
}

// Validate validates a proto message.
func (val *Validator) Validate(msg proto.Message) error {
	return val.v.Validate(msg)
}
