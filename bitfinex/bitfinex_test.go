package bitfinex

import (
	"math"
	"os"
	"testing"
)

var (
	apiPublic  = New("", "")
	apiPrivate = New(os.Getenv("BITFINEX_KEY"), os.Getenv("BITFINEX_SECRET"))
)

func TestTrades(t *testing.T) {
	// Test good request
	trades, err := apiPublic.Trades("ltcusd", 10)
	if err != nil || trades == nil {
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
	book, err := apiPublic.Orderbook("ltcusd", 10, 10)
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
	trades, err := apiPublic.Trades("ltcusd", 1)
	if err != nil {
		t.Error("Failed: " + err.Error())
		return
	}
	// Set a safe sell price above the current price
	price := trades[0].Price + 0.20
	symbol := "ltcusd"
	amount := 0.1
	exchange := "bitfinex"
	side := "sell"
	otype := "limit"

	// Test submitting a new order
	order, err := apiPrivate.NewOrder(symbol, amount, price, exchange, side, otype)
	if err != nil || order.ID == 0 {
		t.Error("Failed : " + err.Error())
		return
	}
	t.Logf("Placed a new sell order of 0.1 ltcusd @ %v limit with ID: %d", price, order.ID)
	if order.Symbol != symbol {
		t.Error("Symbol does not match")
		return
	}
	if math.Abs(order.OriginalAmount-amount) > 0.000001 {
		t.Error("Amount does not match")
		return
	}
	if math.Abs(order.Price-price) > 0.000001 {
		t.Error("Price does not match")
		return
	}
	if order.Exchange != exchange {
		t.Error("Exchange does not match")
		return
	}
	if order.Side != side {
		t.Error("Side does not match")
		return
	}
	if order.Type != otype {
		t.Error("Type does not match")
		return
	}

	// Test status
	order, err = apiPrivate.OrderStatus(order.ID)
	if err != nil {
		t.Error("Failed: " + err.Error())
		return
	}
	if !order.IsLive {
		t.Error("Failed: new order should still be live")
		return
	}
	t.Logf("Order is confirmed live")

	// Test replacing the active order
	price += 0.1
	order, err = apiPrivate.ReplaceOrder(order.ID, symbol, amount, price, exchange, side, otype)
	if err != nil || order.ID == 0 {
		t.Error("Failed: " + err.Error())
		return
	}
	if math.Abs(order.Price-price) > 0.000001 {
		t.Error("Failed: price does not match after attempted replace")
		return
	}
	t.Logf("Increased price by 0.1")

	// Test status
	order, err = apiPrivate.OrderStatus(order.ID)
	if err != nil {
		t.Error("Failed: " + err.Error())
		return
	}
	if !order.IsLive {
		t.Error("Failed: replaced order should still be live")
		return
	}
	t.Logf("Order is confirmed live")

	// Test cancelling the order
	order, err = apiPrivate.CancelOrder(order.ID)
	if err != nil {
		t.Error("Failed: " + err.Error())
		return
	}
	t.Logf("Cancelled order")

	// Test status
	order, err = apiPrivate.OrderStatus(order.ID)
	if err != nil {
		t.Error("Failed: " + err.Error())
		return
	}
	if !order.IsCancelled {
		t.Error("Failed: ORDER NOT CANCELLED!")
		return
	}
	t.Logf("Cancellation is confirmed")

	// Test submitting a bad order
	order, err = apiPrivate.NewOrder("badsymbol", 0.1, 300, "bitfinex", "sell", "limit")
	if order.ID != 0 {
		t.Error("Failed: expected order.ID == 0 on bad order")
		return
	}
}

func TestMultipleNewOrders(t *testing.T) {
	// Get a current price to use for trade
	trades, err := apiPublic.Trades("ltcusd", 1)
	if err != nil {
		t.Error("Failed: " + err.Error())
		return
	}
	// Set safe trade prices
	bidPrice := trades[0].Price - 0.20
	askPrice := trades[0].Price + 0.20

	params := []OrderParams{
		{"ltcusd", 0.1, bidPrice, "bitfinex", "buy", "limit"},
		{"ltcusd", 0.1, askPrice, "bitfinex", "sell", "limit"},
	}

	// Test submitting a new multiple order
	orders, err := apiPrivate.MultipleNewOrders(params)
	if err != nil || orders.Orders[0].ID == 0 || orders.Orders[1].ID == 0 {
		t.Error("Failed: " + err.Error())
		return
	}
	t.Logf("Placed a new buy order of 0.1 ltcusd @ %v limit with ID: %d", bidPrice, orders.Orders[0].ID)
	t.Logf("Placed a new sell order of 0.1 ltcusd @ %v limit with ID: %d", askPrice, orders.Orders[1].ID)

	// Test cancelling all active orders
	success, err := apiPrivate.CancelAll()
	if err != nil || !success {
		t.Error("Failed: " + err.Error())
	}
	t.Logf("Cancelled all active orders")
}
