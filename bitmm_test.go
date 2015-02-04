package main

import (
	"bitmm/bitfinex"
	"code.google.com/p/gcfg"
	// "github.com/davecgh/go-spew/spew"
	"testing"
)

func TestCalcOrderParams(t *testing.T) {
	// Get config info
	err := gcfg.ReadFileInto(&cfg, "bitmm.gcfg")
	if err != nil {
		t.Fatal(err)
	}

	var params []bitfinex.OrderParams
	// Test long postions
	params = calculateOrderParams(cfg.Sec.MaxPos-cfg.Sec.MinPos/2, 2.00, 0.04)
	// spew.Dump(params)
	if len(params) != 1 {
		t.Fatal("Should only create one order")
	}
	params = calculateOrderParams(cfg.Sec.MaxPos-cfg.Sec.MinPos*2, 2.00, 0.01)
	// spew.Dump(params)
	if len(params) != 3 {
		t.Fatal("Should create three orders")
	}
	params = calculateOrderParams(cfg.Sec.MaxPos*2, 2.00, 0.04)
	// spew.Dump(params)
	if len(params) != 1 {
		t.Fatal("Should only create one order")
	}
	params = calculateOrderParams(cfg.Sec.MinPos/2, 2.00, 0.04)
	// spew.Dump(params)
	if len(params) != 2 {
		t.Fatal("Should create two orders")
	}
	// Test short positions
	params = calculateOrderParams(-cfg.Sec.MaxPos+cfg.Sec.MinPos/2, 2.00, 0.04)
	// spew.Dump(params)
	if len(params) != 1 {
		t.Fatal("Should only create one order")
	}
	params = calculateOrderParams(-cfg.Sec.MaxPos+cfg.Sec.MinPos*2, 2.00, 0.04)
	// spew.Dump(params)
	if len(params) != 3 {
		t.Fatal("Should create three orders")
	}
	params = calculateOrderParams(-cfg.Sec.MaxPos*2, 2.00, 0.04)
	// spew.Dump(params)
	if len(params) != 1 {
		t.Fatal("Should only create one order")
	}
	params = calculateOrderParams(-cfg.Sec.MinPos/2, 2.00, 0.04)
	// spew.Dump(params)
	if len(params) != 2 {
		t.Fatal("Should create two orders")
	}
}
