package qschema

import "fmt"

type Storage struct {
	name   string
	volume int

	lastTickNum int
	moveTickNum int
	numOrders   int
	orders      []*Order
	receiver    Element
}

func NewStorage(name string, volume int, receiver Element) *Storage {
	return &Storage{
		name:     name,
		volume:   volume,
		receiver: receiver,
		orders:   []*Order{},
	}
}

func (s *Storage) Report() string {
	report := ""
	if s.receiver != nil {
		report = s.receiver.Report()
	}
	return fmt.Sprintf("Накопитель %s: last tick: %d, orders: %d, заявок в накопителе: %d\n%s",
		s.name, s.lastTickNum, s.numOrders, len(s.orders), report)
}

func (s *Storage) Tick(tickNum int, order *Order) error {
	if tickNum < s.lastTickNum {
		return fmt.Errorf("накопитель %s: номер tick меньше уже обработанного", s.name)
	}

	s.lastTickNum = tickNum
	if order != nil {
		s.orders = append(s.orders, order)
	}

	if s.moveTickNum < tickNum && len(s.orders) > 0 {
		if s.receiver == nil {
			s.moveTickNum = tickNum
			s.numOrders++
			s.orders[0] = nil
			s.orders = s.orders[1:]
			return nil
		}

		if s.receiver.CanAccept(tickNum) {
			order = s.orders[0]
			s.moveTickNum = tickNum
			if err := s.receiver.Tick(tickNum, order); err != nil {
				return fmt.Errorf("накопитель %s: %w", s.name, err)
			}
			s.numOrders++
			s.orders[0] = nil
			s.orders = s.orders[1:]
			return nil
		}
	}

	if s.receiver != nil {
		if err := s.receiver.Tick(tickNum, nil); err != nil {
			return fmt.Errorf("накопитель %s: %w", s.name, err)
		}
	}

	if len(s.orders) > s.volume {
		return fmt.Errorf("накопитель %s: переполнен (%w)", s.name, ErrBlockingScheme)
	}
	return nil
}

func (s *Storage) CanAccept(tickNum int) bool {
	if s.receiver == nil {
		return tickNum > s.lastTickNum && len(s.orders) < s.volume
	}
	if len(s.orders) >= s.volume {
		return tickNum > s.lastTickNum && s.receiver.CanAccept(tickNum)
	}
	return true
}
