package main

import (
	"fmt"
	"os"

	"github.com/NovikovRoman/qschema"
)

func main() {
	ch4 := qschema.NewChannel("K4", 2, nil)
	ch5 := qschema.NewChannel("K5", 2, nil)

	st1 := qschema.NewStorage("H1", 1, ch4)
	st2 := qschema.NewStorage("H2", 1, ch5)

	// --- Ветка 1 ---
	bl4 := qschema.NewBlocker("B4", func(o *qschema.Order, num int) bool {
		return num%2 == 0
	}, st1)
	bl5 := qschema.NewBlocker("B5", func(o *qschema.Order, num int) bool {
		return num%2 != 0
	}, st2)
	bl6 := qschema.NewBlocker("B6", func(o *qschema.Order, num int) bool {
		return true
	}, nil)
	blockers1 := qschema.NewBlockers("BK1", bl4, bl5, bl6)
	ch1 := qschema.NewChannel("K1", 4, blockers1)

	// --- Ветка 2 ---
	bl7 := qschema.NewBlocker("B7", func(o *qschema.Order, num int) bool {
		return num%2 == 0
	}, st1)
	bl8 := qschema.NewBlocker("B8", func(o *qschema.Order, num int) bool {
		return num%2 != 0
	}, st2)
	bl9 := qschema.NewBlocker("B9", func(o *qschema.Order, num int) bool {
		return true
	}, nil)
	blockers2 := qschema.NewBlockers("BK2", bl7, bl8, bl9)
	ch2 := qschema.NewChannel("K2", 5, blockers2)

	// --- Ветка 3 ---
	bl10 := qschema.NewBlocker("B10", func(o *qschema.Order, num int) bool {
		return num%2 == 0
	}, st1)
	bl11 := qschema.NewBlocker("B11", func(o *qschema.Order, num int) bool {
		return num%2 != 0
	}, st2)
	bl12 := qschema.NewBlocker("B12", func(o *qschema.Order, num int) bool {
		return true
	}, nil)
	blockers3 := qschema.NewBlockers("BK3", bl10, bl11, bl12)
	ch3 := qschema.NewChannel("K3", 5, blockers3)

	// --- Блокираторы источника 1 ---
	bl1 := qschema.NewBlocker("B1", nil, ch1)
	bl1_2 := qschema.NewBlocker("B1_2", func(o *qschema.Order, num int) bool {
		return true
	}, nil)
	sourceBlockers1 := qschema.NewBlockers("BI1", bl1, bl1_2)

	// --- Блокираторы источника 2 ---
	bl2 := qschema.NewBlocker("B2", nil, ch2)
	bl2_2 := qschema.NewBlocker("B2_2", func(o *qschema.Order, num int) bool {
		return true
	}, nil)
	sourceBlockers2 := qschema.NewBlockers("BI2", bl2, bl2_2)

	// --- Блокираторы источника 3 ---
	bl3 := qschema.NewBlocker("B3", nil, ch3)
	bl3_2 := qschema.NewBlocker("B3_2", func(o *qschema.Order, num int) bool {
		return true
	}, nil)
	sourceBlockers3 := qschema.NewBlockers("BI3", bl3, bl3_2)

	source1 := qschema.NewSource("I1", 2, sourceBlockers1)
	source2 := qschema.NewSource("I2", 2, sourceBlockers2)
	source3 := qschema.NewSource("I3", 2, sourceBlockers3)

	schema := qschema.New(source1, source2, source3)
	if err := schema.Run(10); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println(schema.Report())
}
