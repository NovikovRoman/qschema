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
	elements    []Element
	lastTickNum int
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

// Run продвигает схему на указанное количество тиков. Если элемент возвращает
// ошибку, частично обработанный тик не повторяется при следующем вызове.
func (s *Schema) Run(ticks int) error {
	for range ticks {
		s.lastTickNum++
		for _, el := range s.elements {
			if err := el.Tick(s.lastTickNum, nil); err != nil {
				return err
			}
		}
	}
	return nil
}
