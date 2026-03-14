package qschema

import (
	"errors"
	"strings"
)

var (
	ErrBlockingScheme = errors.New("блокирование схемы")
)

type Order struct {
	ID string
}

type Element interface {
	Tick(num int, o *Order) error
	CanAccept(tickNum int) bool
	Report() string
}

type Schema struct {
	elements []Element
}

func New(elements ...Element) *Schema {
	return &Schema{
		elements: elements,
	}
}

func (s *Schema) Report() string {
	reports := make([]string, len(s.elements))
	for i, el := range s.elements {
		reports[i] = el.Report()
	}
	return strings.Join(reports, "\n\n")
}

func (s *Schema) Run(ticks int) error {
	for n := 1; n <= ticks; n++ {
		for _, el := range s.elements {
			if err := el.Tick(n, nil); err != nil {
				return err
			}
		}
	}
	return nil
}
