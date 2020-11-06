package flagvar

import (
	"fmt"
	"strings"
)

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

	return fmt.Errorf("must be one of %v", e.Choices)
}

func (e *Enum) String() string {
	if e == nil {
		return ""
	}

	return e.Value
}
