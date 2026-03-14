package qschema

import "fmt"

type Source struct {
	name     string
	ticks    int
	receiver Element

	lastTickNum int
	numOrders   int
	countdown   int
}

func NewSource(name string, ticks int, receiver Element) *Source {
	return &Source{
		name:     name,
		ticks:    ticks,
		receiver: receiver,
	}
}

func (s *Source) Report() string {
	report := ""
	if s.receiver != nil {
		report = s.receiver.Report()
	}
	return fmt.Sprintf("Источник %s: last tick: %d, orders: %d\n%s",
		s.name, s.lastTickNum, s.numOrders, report)
}

func (s *Source) Tick(tickNum int, _ *Order) error {
	if tickNum < s.lastTickNum {
		return fmt.Errorf("источник %s: номер tick %d меньше уже обработанного", s.name, tickNum)
	}
	if s.receiver == nil {
		return fmt.Errorf("tick %d источник %s: нет приемника", tickNum, s.name)
	}
	if tickNum == s.lastTickNum {
		return nil
	}
	s.lastTickNum = tickNum
	s.countdown--
	if s.countdown > 0 {
		return s.receiver.Tick(tickNum, nil)
	}

	s.countdown = s.ticks
	s.numOrders++
	return s.receiver.Tick(tickNum, &Order{ID: fmt.Sprintf("%s:%d", s.name, s.numOrders)})
}

func (s *Source) CanAccept(tickNum int) bool {
	return false
}
