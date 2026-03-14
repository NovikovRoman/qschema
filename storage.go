package qschema

import "fmt"

// orderQueue ring buffer фиксированной емкости.
// Исключает неограниченный рост backing array.
type orderQueue struct {
	buf  []*Order
	head int
	n    int
}

func newOrderQueue(cap int) orderQueue {
	return orderQueue{buf: make([]*Order, cap)}
}

func (q *orderQueue) push(o *Order) {
	q.buf[(q.head+q.n)%len(q.buf)] = o
	q.n++
}

func (q *orderQueue) front() *Order {
	return q.buf[q.head]
}

func (q *orderQueue) pop() {
	q.buf[q.head] = nil
	q.head = (q.head + 1) % len(q.buf)
	q.n--
}

func (q *orderQueue) len() int { return q.n }

type Storage struct {
	name   string
	volume int

	lastTickNum int
	moveTickNum int
	numOrders   int
	orders      orderQueue
	receiver    Element
}

func NewStorage(name string, volume int, receiver Element) *Storage {
	return &Storage{
		name:     name,
		volume:   volume,
		receiver: receiver,
		// volume+1, тк проверка переполнения выполняется после добавления.
		orders: newOrderQueue(volume + 1),
	}
}

func (s *Storage) Report() string {
	report := ""
	if s.receiver != nil {
		report = s.receiver.Report()
	}
	return fmt.Sprintf("Накопитель %s: last tick: %d, orders: %d, заявок в накопителе: %d\n%s",
		s.name, s.lastTickNum, s.numOrders, s.orders.len(), report)
}

func (s *Storage) Tick(tickNum int, order *Order) error {
	if tickNum < s.lastTickNum {
		return fmt.Errorf("накопитель %s: номер tick меньше уже обработанного", s.name)
	}

	if order != nil {
		s.orders.push(order)
	}
	s.lastTickNum = tickNum

	if s.moveTickNum < tickNum && s.orders.len() > 0 { // можно пробросить из хранилища
		if s.receiver == nil {
			s.moveTickNum = tickNum
			s.numOrders++
			s.orders.pop()
			return nil
		}

		if s.receiver.CanAccept(tickNum) {
			order = s.orders.front()
			s.moveTickNum = tickNum
			if err := s.receiver.Tick(tickNum, order); err != nil {
				return fmt.Errorf("накопитель %s: %w", s.name, err)
			}
			s.numOrders++
			s.orders.pop()
			return nil
		}
	}

	if s.receiver != nil {
		if err := s.receiver.Tick(tickNum, nil); err != nil {
			return fmt.Errorf("накопитель %s: %w", s.name, err)
		}
	}

	if s.orders.len() > s.volume {
		return fmt.Errorf("накопитель %s: переполнен (%w)", s.name, ErrBlockingScheme)
	}
	return nil
}

func (s *Storage) CanAccept(tickNum int) bool {
	if s.receiver == nil {
		return tickNum > s.lastTickNum && s.orders.len() < s.volume
	}
	if s.orders.len() >= s.volume {
		return tickNum > s.lastTickNum && s.receiver.CanAccept(tickNum)
	}
	return true
}
