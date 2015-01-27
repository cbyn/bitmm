package main

import (
	"bitmm/bitfinex"
	"testing"
)

func TestCalcOrderParams(t *testing.T) {
	MAXO = 100
	INEDGE = 0.05
	OUTEDGE = 0.01
	var params []bitfinex.OrderParams
	// Test long postions
	params = calcOrderParams(100-MINO/2, 2.00)
	if len(params) != 1 {
		t.Fatal("Should only create one order")
	}
	params = calcOrderParams(100-MINO*2, 2.00)
	if len(params) != 3 {
		t.Fatal("Should create three orders")
	}
	params = calcOrderParams(100*2, 2.00)
	if len(params) != 1 {
		t.Fatal("Should only create one order")
	}
	params = calcOrderParams(MINO/2, 2.00)
	if len(params) != 2 {
		t.Fatal("Should create two orders")
	}
	// Test short positions
	params = calcOrderParams(-100+MINO/2, 2.00)
	if len(params) != 1 {
		t.Fatal("Should only create one order")
	}
	params = calcOrderParams(-100+MINO*2, 2.00)
	if len(params) != 3 {
		t.Fatal("Should create three orders")
	}
	params = calcOrderParams(-100*2, 2.00)
	if len(params) != 1 {
		t.Fatal("Should only create one order")
	}
	params = calcOrderParams(-MINO/2, 2.00)
	if len(params) != 2 {
		t.Fatal("Should create two orders")
	}
}
