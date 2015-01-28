package main

import (
	"bitmm/bitfinex"
	// "github.com/davecgh/go-spew/spew"
	"testing"
)

func TestCalcOrderParams(t *testing.T) {
	maxPos = 100
	minEdge = 0.02
	exitPercent = 0.25
	var params []bitfinex.OrderParams
	// Test long postions
	params = calculateOrderParams(100-MINO/2, 2.00, 0.04)
	// spew.Dump(params)
	if len(params) != 1 {
		t.Fatal("Should only create one order")
	}
	params = calculateOrderParams(100-MINO*2, 2.00, 0.01)
	// spew.Dump(params)
	if len(params) != 3 {
		t.Fatal("Should create three orders")
	}
	params = calculateOrderParams(100*2, 2.00, 0.04)
	// spew.Dump(params)
	if len(params) != 1 {
		t.Fatal("Should only create one order")
	}
	params = calculateOrderParams(MINO/2, 2.00, 0.04)
	// spew.Dump(params)
	if len(params) != 2 {
		t.Fatal("Should create two orders")
	}
	// Test short positions
	params = calculateOrderParams(-100+MINO/2, 2.00, 0.04)
	// spew.Dump(params)
	if len(params) != 1 {
		t.Fatal("Should only create one order")
	}
	params = calculateOrderParams(-100+MINO*2, 2.00, 0.04)
	// spew.Dump(params)
	if len(params) != 3 {
		t.Fatal("Should create three orders")
	}
	params = calculateOrderParams(-100*2, 2.00, 0.04)
	// spew.Dump(params)
	if len(params) != 1 {
		t.Fatal("Should only create one order")
	}
	params = calculateOrderParams(-MINO/2, 2.00, 0.04)
	// spew.Dump(params)
	if len(params) != 2 {
		t.Fatal("Should create two orders")
	}
}
