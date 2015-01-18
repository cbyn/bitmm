package bitfinex

import (
	// "github.com/davecgh/go-spew/spew"
	"math"
	"os"
	"testing"
)

var (
	api = New(os.Getenv("BITFINEX_KEY"), os.Getenv("BITFINEX_SECRET"))
)

func TestTrades(t *testing.T) {
	// Test good request
	trades, err := api.Trades("ltcusd", 10)
	if err != nil || trades == nil {
		t.Fatal(err)
	}

	// Test bad request
	trades, err = api.Trades("badsymbol", 10)
	if trades != nil {
		t.Fatal("Expected empty trades on bad request")
	}
}

func TestOrderbook(t *testing.T) {
	// Test good request
	book, err := api.Orderbook("ltcusd", 10, 10)
	if err != nil {
		t.Fatal(err)
	}
	if book.Bids == nil || book.Asks == nil {
		t.Fatal("Expected non-empty orderbook on good request")
	}

	// Test bad request
	book, err = api.Orderbook("badsymbol", 10, 10)
	if book.Bids != nil || book.Asks != nil {
		t.Fatal("Expected empty orderbook on bad request")
	}
}

func TestNewOrder(t *testing.T) {
	// Get a current price to use for trade
	trades, err := api.Trades("ltcusd", 1)
	if err != nil {
		t.Fatal(err)
	}
	// Set a safe sell price above the current price
	price := trades[0].Price + 0.20
	symbol := "ltcusd"
	amount := 0.1
	exchange := "bitfinex"
	side := "sell"
	otype := "limit"

	// Test submitting a new order
	order, err := api.NewOrder(symbol, amount, price, exchange, side, otype)
	if err != nil || order.ID == 0 {
		t.Fatal(err)
	}
	t.Logf("Placed a new sell order of 0.1 ltcusd @ %v limit with ID: %d", price, order.ID)
	if order.Symbol != symbol {
		t.Fatal("Symbol does not match")
	}
	if math.Abs(order.OriginalAmount-amount) > 0.000001 {
		t.Fatal("Amount does not match")
	}
	if math.Abs(order.Price-price) > 0.000001 {
		t.Fatal("Price does not match")
	}
	if order.Exchange != exchange {
		t.Fatal("Exchange does not match")
	}
	if order.Side != side {
		t.Fatal("Side does not match")
	}
	if order.Type != otype {
		t.Fatal("Type does not match")
	}

	// Test status
	order, err = api.OrderStatus(order.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !order.IsLive {
		t.Fatal("New order should still be live")
	}
	t.Logf("Order is confirmed live")

	// Test replacing the active order
	price += 0.1
	order, err = api.ReplaceOrder(order.ID, symbol, amount, price, exchange, side, otype)
	if err != nil || order.ID == 0 {
		t.Fatal(err)
	}
	if math.Abs(order.Price-price) > 0.000001 {
		t.Fatal("Price does not match after attempted replace")
	}
	t.Logf("Increased price by 0.1")

	// Test status
	order, err = api.OrderStatus(order.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !order.IsLive {
		t.Fatal("Replaced order should still be live")
	}
	t.Logf("Order is confirmed live")

	// Test cancelling the order
	order, err = api.CancelOrder(order.ID)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Cancelled order")

	// Test status
	order, err = api.OrderStatus(order.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !order.IsCancelled {
		t.Fatal("ORDER NOT CANCELLED!")
	}
	t.Logf("Cancellation is confirmed")

	// Test submitting a bad order
	order, err = api.NewOrder("badsymbol", 0.1, 300, "bitfinex", "sell", "limit")
	if order.ID != 0 {
		t.Fatal("Expected order.ID == 0 on bad order")
	}
}

func TestMultipleNewOrders(t *testing.T) {
	// Get a current price to use for trade
	trades, err := api.Trades("ltcusd", 1)
	if err != nil {
		t.Fatal(err)
	}
	// Set safe trade prices
	bidPrice := trades[0].Price - 0.20
	askPrice := trades[0].Price + 0.20

	params := []OrderParams{
		{"ltcusd", 0.1, bidPrice, "bitfinex", "buy", "limit"},
		{"ltcusd", 0.1, askPrice, "bitfinex", "sell", "limit"},
	}

	// Test submitting a new multiple order
	orders, err := api.MultipleNewOrders(params)
	if err != nil || orders.Orders[0].ID == 0 || orders.Orders[1].ID == 0 {
		t.Fatal(err)
	}
	t.Logf("Placed a new buy order of 0.1 ltcusd @ %v limit with ID: %d", bidPrice, orders.Orders[0].ID)
	t.Logf("Placed a new sell order of 0.1 ltcusd @ %v limit with ID: %d", askPrice, orders.Orders[1].ID)

	// Test cancelling all active orders
	success, err := api.CancelAll()
	if err != nil || !success {
		t.Fatal(err)
	}
	t.Logf("Cancelled all active orders")
}

func TestActivePositions(t *testing.T) {
	_, err := api.ActivePositions()
	if err != nil {
		t.Fatal(err)
	}
}
