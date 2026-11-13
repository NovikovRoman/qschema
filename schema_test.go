package qschema

import (
	"errors"
	"testing"
)

type testReceiver struct {
	accept bool
	err    error
	ticks  []int
	orders []*Order
}

func (r *testReceiver) Tick(tick int, order *Order) error {
	r.ticks = append(r.ticks, tick)
	if r.err != nil {
		return r.err
	}
	if order != nil {
		r.orders = append(r.orders, order)
	}
	return nil
}
func (r *testReceiver) CanAccept(int) bool { return r.accept }
func (r *testReceiver) Report() string     { return "" }

func TestSchemaRunContinuesTicks(t *testing.T) {
	r := &testReceiver{accept: true}
	s := New(r)
	if err := s.Run(2); err != nil {
		t.Fatal(err)
	}
	if err := s.Run(2); err != nil {
		t.Fatal(err)
	}
	for i, tick := range r.ticks {
		if tick != i+1 {
			t.Fatalf("такты: %v", r.ticks)
		}
	}
	if len(r.ticks) != 4 {
		t.Fatalf("такты: %v", r.ticks)
	}
}

func TestChannelRetainsOrderOnReceiverError(t *testing.T) {
	for _, replacement := range []bool{false, true} {
		t.Run(map[bool]string{false: "пустой такт", true: "замена"}[replacement], func(t *testing.T) {
			r := &testReceiver{accept: true}
			c := NewChannel("channel", 1, r)
			first := &Order{ID: "first"}
			if err := c.Tick(1, first); err != nil {
				t.Fatal(err)
			}
			failure := errors.New("ошибка получателя")
			r.err = failure
			var next *Order
			if replacement {
				next = &Order{ID: "second"}
			}
			if err := c.Tick(2, next); !errors.Is(err, failure) {
				t.Fatalf("получено: %v", err)
			}
			if c.order != first || c.numOrders != 0 {
				t.Fatalf("заявка=%v количество=%d", c.order, c.numOrders)
			}
			r.err = nil
			if err := c.Tick(2, next); err != nil {
				t.Fatal(err)
			}
			if len(r.orders) != 1 || r.orders[0] != first {
				t.Fatalf("доставлено: %v", r.orders)
			}
		})
	}
}

func TestChannelCanAcceptAfterSameTickFailure(t *testing.T) {
	r := &testReceiver{accept: true}
	c := NewChannel("channel", 1, r)
	if err := c.Tick(1, &Order{ID: "first"}); err != nil {
		t.Fatal(err)
	}
	r.err = errors.New("ошибка")
	if err := c.Tick(2, nil); err == nil {
		t.Fatal("ожидалась ошибка")
	}
	if !c.CanAccept(2) {
		t.Fatal("готовый канал должен принимать заявки на том же такте")
	}
}

func TestStorageInvalidCapacity(t *testing.T) {
	for _, capacity := range []int{-2, -1, int(^uint(0) >> 1)} {
		s := NewStorage("storage", capacity, nil)
		if s.CanAccept(1) {
			t.Fatalf("емкость %d принята", capacity)
		}
		if err := s.Tick(1, &Order{}); err == nil {
			t.Fatalf("емкость %d: ожидалась ошибка", capacity)
		}
	}
}

func TestStorageOverflowDoesNotCorruptQueue(t *testing.T) {
	r := &testReceiver{accept: false}
	s := NewStorage("storage", 1, r)
	first := &Order{ID: "first"}
	if err := s.Tick(1, first); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if err := s.Tick(1, &Order{ID: "overflow"}); !errors.Is(err, ErrBlockingScheme) {
			t.Fatalf("получено: %v", err)
		}
		if s.orders.len() != 1 || s.orders.front() != first {
			t.Fatal("очередь изменилась после переполнения")
		}
	}
	r.accept = true
	if err := s.Tick(2, nil); err != nil {
		t.Fatal(err)
	}
	if len(r.orders) != 1 || r.orders[0] != first {
		t.Fatalf("доставлено: %v", r.orders)
	}
}

func TestStorageZeroCapacityDirectTransfer(t *testing.T) {
	s := NewStorage("storage", 0, nil)
	if !s.CanAccept(1) {
		t.Fatal("ожидалась возможность прямой передачи")
	}
	if err := s.Tick(1, &Order{}); err != nil {
		t.Fatal(err)
	}
	if s.CanAccept(1) {
		t.Fatal("нельзя передавать дважды за один такт")
	}
}
