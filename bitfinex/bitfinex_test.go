package bitfinex

import (
	"os"
	"testing"
)

var APIKey = os.Getenv("BITFINEX_KEY")
var APISecret = os.Getenv("BITFINEX_SECRET")

var apiPublic = New("", "")
var apiPrivate = New(APIKey, APISecret)

func TestTrades(t *testing.T) {
	// Test good request
	trades, err := apiPublic.Trades("btcusd", 10)
	if err != nil {
		t.Error("Failed: " + err.Error())
		return
	}

	// Test bad request
	trades, err = apiPublic.Trades("badsymbol", 10)
	if trades != nil {
		t.Error("Failed: expected empty trades on bad request")
		return
	}
}

func TestOrderbook(t *testing.T) {
	// Test good request
	book, err := apiPublic.Orderbook("btcusd", 10, 10)
	if err != nil {
		t.Error("Failed: " + err.Error())
		return
	}
	if book.Bids == nil || book.Asks == nil {
		t.Error("Failed: expected non-empty orderbook on good request")
		return
	}

	// Test bad request
	book, err = apiPublic.Orderbook("badsymbol", 10, 10)
	if book.Bids != nil || book.Asks != nil {
		t.Error("Failed: expected empty orderbook on bad request")
		return
	}
}

func TestNewOrder(t *testing.T) {
	// Get a current price to use for trade
	trades, err := apiPublic.Trades("btcusd", 1)
	if err != nil {
		t.Error("Failed: " + err.Error())
		return
	}
	// Safe sell price
	price := trades[0].Price + 100
	t.Logf("Using trade price: %v", price)

	// Test good order
	order, err := apiPrivate.NewOrder("btcusd", 0.1, price, "bitfinex", "sell", "limit", true)
	if err != nil || order.ID == 0 {
		t.Error("Failed: " + err.Error())
		return
	}
	t.Logf("Placed a new sell order of 0.1 btcusd @ 300 limit with ID: %d", order.ID)

	// Test active orders
	orders, err := apiPrivate.ActiveOrders()
	if err != nil {
		t.Error("Failed: " + err.Error())
		return
	}
	if orders[0] != order {
		t.Error("Failed: expected active orders to return current order")
		return
	}

	// Test status

	// Test replace

	// Test cancel

	// Test bad order
	order, err = apiPrivate.NewOrder("badsymbol", 0.1, 300, "bitfinex", "sell", "limit", true)
	if err == nil {
		t.Error("Failed: expected error on bad order")
		return
	}
}
