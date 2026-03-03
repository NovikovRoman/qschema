package qschema

import "fmt"

type Channel struct {
	name     string
	ticks    int
	receiver Element

	lastTickNum int
	countdown   int
	order       *Order
	numOrders   int
}

func NewChannel(name string, ticks int, receiver Element) *Channel {
	return &Channel{
		name:     name,
		ticks:    ticks,
		receiver: receiver,
	}
}

func (c *Channel) Report() string {
	report := ""
	if c.receiver != nil {
		report = c.receiver.Report()
	}
	orderNum := "-"
	if c.order != nil {
		orderNum = fmt.Sprintf("#%d", c.order.Num)
	}
	return fmt.Sprintf("Канал %s: last tick: %d, orders: %d, на обработке заявка: %s\n%s",
		c.name, c.lastTickNum, c.numOrders, orderNum, report)
}

func (c *Channel) Tick(tickNum int, o *Order) error {
	if tickNum < c.lastTickNum {
		return fmt.Errorf("канал %s: номер tick меньше уже обработанного", c.name)
	}

	if c.lastTickNum < tickNum {
		c.lastTickNum = tickNum
		if c.order != nil {
			c.countdown--
		}
	}

	if o == nil {
		if c.countdown <= 0 {
			o = c.order
			c.order = nil
			if o != nil {
				c.numOrders++
			}
		}
		if c.receiver == nil {
			return nil
		}
		return c.receiver.Tick(tickNum, o)
	}

	if c.order == nil {
		c.order = o
		c.countdown = c.ticks
		if c.receiver == nil {
			return nil
		}
		return c.receiver.Tick(tickNum, nil)
	}

	if c.countdown > 0 {
		return fmt.Errorf("канал %s: данные еще на обработке (%w)", c.name, ErrBlockingScheme)
	}

	// проверить дальше по схеме
	if c.receiver == nil {
		c.order = o
		c.countdown = c.ticks
		return nil
	}
	if !c.receiver.CanAccept(tickNum) {
		return fmt.Errorf("канал %s: данные еще на обработке (%w)", c.name, ErrBlockingScheme)
	}

	channelOrder := c.order
	c.order = o
	c.countdown = c.ticks
	c.numOrders++
	return c.receiver.Tick(tickNum, channelOrder)
}

func (c *Channel) CanAccept(tickNum int) bool {
	if c.order == nil {
		return true
	}
	// рассчитать пора ли отправлять дальше
	canSend := tickNum > c.lastTickNum && c.countdown <= 1
	if !canSend {
		return false
	}
	if c.receiver == nil {
		return true
	}
	return c.receiver.CanAccept(tickNum)
}
