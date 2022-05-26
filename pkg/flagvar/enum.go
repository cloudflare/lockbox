package flagvar

import (
	"errors"
	"fmt"
	"strings"
)

var ErrInvalidEnum = errors.New("invalid enum option")

type Enum struct {
	Choices []string
	Value   string
}

func (e *Enum) Help() string {
	return fmt.Sprintf("one of %v", e.Choices)
}

func (e *Enum) Set(v string) error {
	for _, c := range e.Choices {
		if strings.EqualFold(c, v) {
			e.Value = strings.ToLower(v)
			return nil
		}
	}

	return ErrInvalidEnum
}

func (e *Enum) String() string {
	if e == nil {
		return ""
	}

	return e.Value
}
