package main

import (
	"fmt"
	"os"

	"github.com/NovikovRoman/qschema"
)

func main() {
	ch4 := qschema.NewChannel("К4", 5, nil)
	ch5 := qschema.NewChannel("К5", 5, nil)

	st1 := qschema.NewStorage("Н1", 5, ch4)
	st2 := qschema.NewStorage("Н2", 5, ch5)

	bl4 := qschema.NewBlocker("Б4", func(o *qschema.Order, num int) bool {
		return num%2 == 0
	}, st1)
	bl5 := qschema.NewBlocker("Б5", func(o *qschema.Order, num int) bool {
		return num%2 != 0
	}, st2)
	bl6 := qschema.NewBlocker("Б6", func(o *qschema.Order, num int) bool {
		return true
	}, nil)
	blockers := qschema.NewBlockers("БK1", bl4, bl5, bl6)

	ch1 := qschema.NewChannel("К1", 2, blockers)

	bl1 := qschema.NewBlocker("Б1", nil, ch1)
	bl2 := qschema.NewBlocker("Б2", func(o *qschema.Order, num int) bool {
		return true
	}, nil)
	sourceBlockers := qschema.NewBlockers("БИ1", bl1, bl2)

	source := qschema.NewSource("И1", 1, sourceBlockers)

	schema := qschema.New(source)
	if err := schema.Run(20); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println(schema.Report())
}
