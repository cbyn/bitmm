package bitfinex

import (
	"math"
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
	order, err := apiPrivate.NewOrder("btcusd", 0.1, price, "bitfinex", "sell", "limit")
	if err != nil || order.ID == 0 {
		t.Error("Failed: " + err.Error())
		return
	}
	t.Logf("Placed a new sell order of 0.1 btcusd @ 300 limit with ID: %d", order.ID)
	if order.Symbol != "btcusd" {
		t.Error("Symbol does not match")
		return
	}
	if math.Abs(order.OriginalAmount-0.1) > 0.000001 {
		t.Error("Amount does not match")
		return
	}
	if math.Abs(order.Price-price) > 0.000001 {
		t.Error("Price does not match")
		return
	}
	if order.Exchange != "bitfinex" {
		t.Error("Exchange does not match")
		return
	}
	if order.Side != "sell" {
		t.Error("Side does not match")
		return
	}
	if order.Type != "limit" {
		t.Error("Type does not match")
		return
	}

	// Test active orders
	orders, err := apiPrivate.ActiveOrders()
	if err != nil {
		t.Error("Failed: " + err.Error())
		return
	}
	if orders[0].ID != order.ID {
		t.Error("Failed: expected active orders to return current order")
		return
	}

	// Test status
	order2, err := apiPrivate.OrderStatus(order.ID)
	if err != nil {
		t.Error("Failed: " + err.Error())
		return
	}
	if order.ID != order2.ID {
		t.Error("Failed: expected order status to return current order")
		return
	}

	// Test replace
	price += 10
	order, err = apiPrivate.ReplaceOrder(order.ID, "btcusd", 0.1, price, "bitfinex", "sell", "limit")
	if err != nil || order.ID == 0 {
		t.Error("Failed: " + err.Error())
		return
	}
	if math.Abs(order.Price-price) > 0.000001 {
		t.Error("Price does not match after attempted replace")
		return
	}
	t.Logf("Increased price by 10 for order ID: %d", order.ID)

	// Test cancel
	order, err = apiPrivate.CancelOrder(order.ID)
	if err != nil || order.ID == 0 {
		t.Error("Failed: " + err.Error())
		return
	}
	if !order.IsCancelled {
		t.Error("***** ORDER NOT CANCELLED! *****")
	}
	t.Logf("Cancelled order with ID: %d", order.ID)

	// Test bad order
	order, err = apiPrivate.NewOrder("badsymbol", 0.1, 300, "bitfinex", "sell", "limit")
	if err == nil {
		t.Error("Failed: expected error on bad order")
		return
	}
}
