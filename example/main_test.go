package main

import (
	"fmt"
	"strings"
	"testing"
)

func TestExampleSchema(t *testing.T) {
	schema := newSchema()
	if err := schema.Run(10); err != nil {
		t.Fatalf("ошибка моделирования: %v", err)
	}

	report := schema.Report()
	for _, expected := range []string{
		"Источник I1: last tick: 10, orders: 5",
		"Источник I2: last tick: 10, orders: 5",
		"Источник I3: last tick: 10, orders: 5",
		"Блокиратор B1: 3",
		"Блокиратор B1_2: 2",
		"Блокиратор B2: 2",
		"Блокиратор B2_2: 3",
		"Блокиратор B3: 2",
		"Блокиратор B3_2: 3",
		"Блокиратор B4: 1",
		"Блокиратор B5: 1",
		"Блокиратор B6: 0",
		"Блокиратор B7: 0",
		"Блокиратор B8: 1",
		"Блокиратор B9: 0",
		"Блокиратор B10: 0",
		"Блокиратор B11: 0",
		"Блокиратор B12: 1",
		"Канал K1: last tick: 10, orders: 2, на обработке заявка: #I1:5",
		"Канал K2: last tick: 10, orders: 1, на обработке заявка: #I2:4",
		"Канал K3: last tick: 10, orders: 1, на обработке заявка: #I3:4",
	} {
		if !strings.Contains(report, expected) {
			t.Errorf("в отчете отсутствует %q", expected)
		}
	}

	// Общие элементы отображаются в отчете каждой из трех веток.
	for _, expected := range []string{
		"Накопитель H1: last tick: 10, orders: 1, заявок в накопителе: 0",
		"Накопитель H2: last tick: 10, orders: 2, заявок в накопителе: 0",
		"Канал K4: last tick: 10, orders: 0, на обработке заявка: #I1:3",
		"Канал K5: last tick: 10, orders: 2, на обработке заявка: -",
	} {
		if count := strings.Count(report, expected); count != 3 {
			t.Errorf("%q встречается %d раз вместо 3", expected, count)
		}
	}
}

func TestExampleSchemaLongRun(t *testing.T) {
	schema := newSchema()
	if err := schema.Run(1000); err != nil {
		t.Fatalf("ошибка длительного моделирования: %v", err)
	}
	for _, source := range []string{"I1", "I2", "I3"} {
		expected := fmt.Sprintf("Источник %s: last tick: 1000, orders: 500", source)
		if !strings.Contains(schema.Report(), expected) {
			t.Errorf("в отчете отсутствует %q", expected)
		}
	}
}

func TestExampleSchemaRunInParts(t *testing.T) {
	whole := newSchema()
	if err := whole.Run(10); err != nil {
		t.Fatal(err)
	}
	for _, parts := range [][]int{{1, 9}, {5, 5}, {1, 2, 3, 4}, {0, 10, 0}} {
		t.Run(fmt.Sprint(parts), func(t *testing.T) {
			chunked := newSchema()
			for _, ticks := range parts {
				if err := chunked.Run(ticks); err != nil {
					t.Fatalf("Run(%d): %v", ticks, err)
				}
			}
			if got, want := chunked.Report(), whole.Report(); got != want {
				t.Errorf("отчет отличается от непрерывного запуска\nполучено:\n%s\nожидалось:\n%s", got, want)
			}
		})
	}
}
