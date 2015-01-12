package bitfinex

import (
	"os"
	"testing"
)

var APIKey = os.Getenv("BITFINEX_KEY")
var APISecret = os.Getenv("BITFINEX_SECRET")

var apiPublic = New("", "")
var apiPrivate = New(APIKey, APISecret)

func TestBook(t *testing.T) {
	// Good request
	book, err := apiPublic.Orderbook("btcusd", 10, 10)
	if err != nil {
		t.Error("Failed: " + err.Error())
		return
	}
	if book.Bids == nil || book.Asks == nil {
		t.Error("Failed: expected non-empty book on good request")
		return
	}

	// Bad request
	book, err = apiPublic.Orderbook("skittles", 10, 10)
	if book.Bids != nil || book.Asks != nil {
		t.Error("Failed: expected empty book on bad request")
		return
	}
}

func TestNewOrder(t *testing.T) {
	// Good order
	order, err := apiPrivate.NewOrder("btcusd", 0.1, 300, "bitfinex", "sell", "limit", true)
	if err != nil || order.ID == 0 {
		t.Error("Failed: " + err.Error())
		return
	}
	t.Logf("Placed a new sell order of 0.1 btcusd @ 300 limit with ID: %d", order.ID)

	// Bad order
	order, err = apiPrivate.NewOrder("apples", 0.1, 300, "bitfinex", "sell", "limit", true)
	if err == nil {
		t.Error("Failed: expected error on bad order")
		return
	}
}
