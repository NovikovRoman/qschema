package qschema

import "fmt"

// orderQueue кольцевой буфер фиксированной емкости.
// Исключает неограниченный рост базового массива.
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

func (q *orderQueue) popBack() {
	index := (q.head + q.n - 1) % len(q.buf)
	q.buf[index] = nil
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

// NewStorage создает накопитель с неотрицательной емкостью.
// При недопустимой емкости Tick возвращает ошибку, а CanAccept false.
func NewStorage(name string, volume int, receiver Element) *Storage {
	capacity := 0
	if volume >= 0 && volume < int(^uint(0)>>1) {
		capacity = volume + 1
	}
	return &Storage{
		name:     name,
		volume:   volume,
		receiver: receiver,
		orders:   newOrderQueue(capacity),
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
	if len(s.orders.buf) == 0 {
		return fmt.Errorf("накопитель %s: недопустимая емкость %d", s.name, s.volume)
	}
	if tickNum < s.lastTickNum {
		return fmt.Errorf("накопитель %s: номер tick меньше уже обработанного", s.name)
	}

	if order != nil {
		s.orders.push(order)
		// Удалить входящую заявку, если после этого тика ей не хватило места.
		defer func() {
			if s.orders.len() > s.volume {
				s.orders.popBack()
			}
		}()
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
	if len(s.orders.buf) == 0 || tickNum < s.lastTickNum {
		return false
	}
	if s.orders.len() < s.volume {
		return true
	}
	return s.moveTickNum < tickNum && (s.receiver == nil || s.receiver.CanAccept(tickNum))
}
