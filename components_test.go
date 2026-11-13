package qschema

import (
	"errors"
	"fmt"
	"reflect"
	"testing"
)

func TestSourceIntervalsAndRepeatedTick(t *testing.T) {
	for _, interval := range []int{1, 2, 3} {
		t.Run(fmt.Sprint(interval), func(t *testing.T) {
			r := &testReceiver{accept: true}
			s := NewSource("S", interval, r)
			for tick := 1; tick <= 7; tick++ {
				if err := s.Tick(tick, nil); err != nil {
					t.Fatal(err)
				}
				if err := s.Tick(tick, nil); err != nil {
					t.Fatal(err)
				}
			}
			var want []string
			for tick := 1; tick <= 7; tick += interval {
				want = append(want, fmt.Sprintf("S:%d", len(want)+1))
			}
			var got []string
			for _, order := range r.orders {
				got = append(got, order.ID)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("заявки: %v, ожидалось %v", got, want)
			}
			if len(r.ticks) != 7 {
				t.Fatalf("повторная обработка тика: %v", r.ticks)
			}
			if err := s.Tick(6, nil); err == nil {
				t.Fatal("ожидалась ошибка старого тика")
			}
			if s.CanAccept(8) {
				t.Fatal("источник не должен принимать заявки")
			}
		})
	}
}

func TestSourceErrors(t *testing.T) {
	if err := NewSource("S", 1, nil).Tick(1, nil); err == nil {
		t.Fatal("ожидалась ошибка отсутствия получателя")
	}
	failure := errors.New("ошибка получателя")
	for _, interval := range []int{1, 2} {
		r := &testReceiver{accept: true}
		s := NewSource("S", interval, r)
		if interval == 2 {
			if err := s.Tick(1, nil); err != nil {
				t.Fatal(err)
			}
		}
		r.err = failure
		if err := s.Tick(interval, nil); !errors.Is(err, failure) {
			t.Fatalf("получено %v", err)
		}
	}
}

func TestBlockersRouting(t *testing.T) {
	first, second := &testReceiver{accept: true}, &testReceiver{accept: true}
	var seen []int
	b := NewBlockers("routes",
		NewBlocker("even", func(_ *Order, n int) bool { seen = append(seen, n); return n%2 == 0 }, first),
		NewBlocker("other", nil, second),
	)
	for i := 1; i <= 3; i++ {
		if err := b.Tick(i, &Order{ID: fmt.Sprint(i)}); err != nil {
			t.Fatal(err)
		}
	}
	if !reflect.DeepEqual(seen, []int{1, 2, 3}) {
		t.Fatalf("номера: %v", seen)
	}
	if len(first.orders) != 1 || first.orders[0].ID != "2" || len(second.orders) != 2 || second.orders[0].ID != "1" || second.orders[1].ID != "3" {
		t.Fatalf("маршрутизация: %v / %v", first.orders, second.orders)
	}
	if !reflect.DeepEqual(first.ticks, []int{1, 2, 3}) || !reflect.DeepEqual(second.ticks, []int{1, 2, 3}) {
		t.Fatal("тики не переданы всем веткам")
	}
	if err := b.Tick(2, nil); err == nil {
		t.Fatal("ожидалась ошибка старого тика")
	}
}

func TestBlockersFallbackAndErrors(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		b := NewBlockers("empty")
		if b.CanAccept(1) {
			t.Fatal("пустая система принимает заявки")
		}
		if err := b.Tick(1, nil); !errors.Is(err, ErrBlockingScheme) {
			t.Fatalf("ошибка: %v", err)
		}
	})
	t.Run("fallback", func(t *testing.T) {
		r := &testReceiver{accept: false}
		fallback := NewBlocker("fallback", nil, nil)
		b := NewBlockers("routes", NewBlocker("busy", nil, r), fallback)
		if !b.CanAccept(1) {
			t.Fatal("наличие веток должно допускать прием")
		}
		if err := b.Tick(1, &Order{ID: "1"}); err != nil {
			t.Fatal(err)
		}
		if len(r.orders) != 0 || fallback.numOrders != 1 {
			t.Fatal("заявка не попала в запасную ветку")
		}
	})
	t.Run("no matching route", func(t *testing.T) {
		b := NewBlockers("routes", NewBlocker("never", func(*Order, int) bool { return false }, nil))
		if err := b.Tick(1, &Order{}); !errors.Is(err, ErrBlockingScheme) {
			t.Fatalf("ошибка: %v", err)
		}
	})
	for _, withOrder := range []bool{false, true} {
		t.Run(fmt.Sprintf("receiver_error_%t", withOrder), func(t *testing.T) {
			failure := errors.New("ошибка получателя")
			b := NewBlockers("routes", NewBlocker("receiver", nil, &testReceiver{accept: true, err: failure}))
			var order *Order
			if withOrder {
				order = &Order{ID: "1"}
			}
			if err := b.Tick(1, order); !errors.Is(err, failure) {
				t.Fatalf("ошибка: %v", err)
			}
		})
	}
}

func TestChannelDelayAndBackpressure(t *testing.T) {
	r := &testReceiver{accept: true}
	c := NewChannel("C", 2, r)
	first, second := &Order{ID: "1"}, &Order{ID: "2"}
	if err := c.Tick(1, first); err != nil {
		t.Fatal(err)
	}
	if err := c.Tick(1, second); !errors.Is(err, ErrBlockingScheme) {
		t.Fatalf("ошибка: %v", err)
	}
	if err := c.Tick(2, nil); err != nil {
		t.Fatal(err)
	}
	if len(r.orders) != 0 || c.CanAccept(2) {
		t.Fatal("заявка передана раньше срока")
	}
	r.accept = false
	if c.CanAccept(3) {
		t.Fatal("получатель занят")
	}
	if err := c.Tick(3, second); !errors.Is(err, ErrBlockingScheme) {
		t.Fatalf("ошибка: %v", err)
	}
	r.accept = true
	if !c.CanAccept(3) {
		t.Fatal("канал готов к передаче")
	}
	if err := c.Tick(3, second); err != nil {
		t.Fatal(err)
	}
	if err := c.Tick(4, nil); err != nil {
		t.Fatal(err)
	}
	if err := c.Tick(5, nil); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(r.orders, []*Order{first, second}) {
		t.Fatalf("порядок заявок: %v", r.orders)
	}
	if err := c.Tick(4, nil); err == nil {
		t.Fatal("ожидалась ошибка старого тика")
	}
}

func TestChannelWithoutReceiver(t *testing.T) {
	c := NewChannel("C", 1, nil)
	if err := c.Tick(1, &Order{ID: "1"}); err != nil {
		t.Fatal(err)
	}
	if !c.CanAccept(2) {
		t.Fatal("готовый канал должен принимать заявку")
	}
	if err := c.Tick(2, &Order{ID: "2"}); err != nil {
		t.Fatal(err)
	}
	if err := c.Tick(3, nil); err != nil {
		t.Fatal(err)
	}
	if c.numOrders != 2 || c.order != nil {
		t.Fatalf("обработано: %d, осталось: %v", c.numOrders, c.order)
	}
}

func TestStorageFIFOAndOneTransferPerTick(t *testing.T) {
	r := &testReceiver{accept: false}
	s := NewStorage("S", 2, r)
	orders := []*Order{{ID: "1"}, {ID: "2"}, {ID: "3"}, {ID: "4"}}
	if err := s.Tick(1, orders[0]); err != nil {
		t.Fatal(err)
	}
	if err := s.Tick(1, orders[1]); err != nil {
		t.Fatal(err)
	}
	r.accept = true
	if err := s.Tick(2, orders[2]); err != nil {
		t.Fatal(err)
	}
	if err := s.Tick(2, nil); err != nil {
		t.Fatal(err)
	}
	if len(r.orders) != 1 {
		t.Fatalf("передано за тик: %d", len(r.orders))
	}
	if err := s.Tick(3, orders[3]); err != nil {
		t.Fatal(err)
	}
	if err := s.Tick(4, nil); err != nil {
		t.Fatal(err)
	}
	if err := s.Tick(5, nil); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(r.orders, orders) {
		t.Fatalf("порядок заявок: %v", r.orders)
	}
	if err := s.Tick(4, nil); err == nil {
		t.Fatal("ожидалась ошибка старого тика")
	}
}

func TestStorageReceiverErrorPreservesQueuedOrder(t *testing.T) {
	r := &testReceiver{accept: false}
	s := NewStorage("S", 1, r)
	order := &Order{ID: "1"}
	if err := s.Tick(1, order); err != nil {
		t.Fatal(err)
	}
	failure := errors.New("ошибка получателя")
	r.accept, r.err = true, failure
	if err := s.Tick(2, nil); !errors.Is(err, failure) {
		t.Fatalf("ошибка: %v", err)
	}
	if s.orders.len() != 1 || s.orders.front() != order {
		t.Fatal("заявка потеряна")
	}
	r.err = nil
	if err := s.Tick(3, nil); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(r.orders, []*Order{order}) {
		t.Fatal("заявка не доставлена")
	}
}

func TestSchemaStopsOnErrorAndContinuesNextTick(t *testing.T) {
	failure := errors.New("ошибка элемента")
	first, failing, last := &testReceiver{}, &testReceiver{err: failure}, &testReceiver{}
	s := New(first, failing, last)
	if err := s.Run(2); !errors.Is(err, failure) {
		t.Fatalf("ошибка: %v", err)
	}
	if len(last.ticks) != 0 {
		t.Fatal("элемент после ошибки был вызван")
	}
	failing.err = nil
	if err := s.Run(1); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first.ticks, []int{1, 2}) || !reflect.DeepEqual(failing.ticks, []int{1, 2}) || !reflect.DeepEqual(last.ticks, []int{2}) {
		t.Fatalf("тики: %v %v %v", first.ticks, failing.ticks, last.ticks)
	}
}
