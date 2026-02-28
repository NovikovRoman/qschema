package qschema

import (
	"fmt"
	"strings"
)

type Blockers struct {
	name  string
	items []*Blocker

	lastTickNum int
	numOrders   int
}

func NewBlockers(name string, items ...*Blocker) *Blockers {
	return &Blockers{
		name:  name,
		items: items,
	}
}

func (b *Blockers) Report() string {
	itemReports := []string{}
	for _, item := range b.items {
		itemReports = append(itemReports, item.Report())
	}
	for _, item := range b.items {
		if item.receiver != nil {
			itemReports = append(itemReports, item.receiver.Report())
		}
	}
	return fmt.Sprintf("Система блокираторов %s: last tick: %d, orders: %d\n%s",
		b.name, b.lastTickNum, b.numOrders, strings.Join(itemReports, "\n"))
}

func (b *Blockers) Tick(tickNum int, o *Order) error {
	if len(b.items) == 0 {
		return fmt.Errorf("система блокираторов %s: %w", b.name, ErrBlockingScheme)
	}
	if tickNum < b.lastTickNum {
		return fmt.Errorf("система блокираторов %s: номер tick меньше уже обработанного", b.name)
	}

	if b.lastTickNum < tickNum {
		b.lastTickNum = tickNum
	}

	if o != nil {
		b.numOrders++
	}

	for _, item := range b.items {
		if o != nil && item.canAccept(tickNum, o, b.numOrders) {
			sendOrder := o
			o = nil
			if err := item.Tick(tickNum, sendOrder); err != nil {
				return fmt.Errorf("блокиратор %s: заявка #%d %w", item.name, sendOrder.Num, err)
			}
			continue
		}

		if err := item.Tick(tickNum, nil); err != nil {
			return fmt.Errorf("блокиратор %s: %w", item.name, err)
		}
	}
	if o != nil {
		return fmt.Errorf("система блокираторов %s: не отправлены данные (%w)", b.name, ErrBlockingScheme)
	}
	return nil
}

// CanAccept проверяет есть ли блокираторы,
// но не обязан проверять готовность получателей.
func (b *Blockers) CanAccept(tickNum int) bool {
	return len(b.items) > 0
}

type Blocker struct {
	name      string
	receiver  Element
	condition func(o *Order, num int) bool

	numOrders int
}

func NewBlocker(name string, condition func(o *Order, num int) bool, receiver Element) *Blocker {
	return &Blocker{
		name:      name,
		condition: condition,
		receiver:  receiver,
	}
}

func (b *Blocker) Report() string {
	return fmt.Sprintf("\tБлокиратор %s: %d", b.name, b.numOrders)
}

func (b *Blocker) Tick(tickNum int, o *Order) error {
	if o != nil {
		b.numOrders++
	}
	if b.receiver == nil {
		return nil
	}
	return b.receiver.Tick(tickNum, o)
}

func (b *Blocker) canAccept(tickNum int, o *Order, numOrders int) bool {
	return (b.condition == nil || b.condition(o, numOrders)) && (b.receiver == nil || b.receiver.CanAccept(tickNum))
}
