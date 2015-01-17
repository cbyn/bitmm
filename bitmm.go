package main

import (
	"./bitfinex"
	"fmt"
	"log"
	"math"
	"os"
	"os/exec"
	"time"
)

// Trade inputs
const (
	SYMBOL    = "ltcusd" // Instrument to trade
	MINCHANGE = 0.0005   // Minumum change required to update prices
	TRADENUM  = 10       // Number of trades to use in calculations
	AMOUNT    = 0.50     // Size to trade
	BIDEDGE   = 0.05     // Required edge for a buy order
	ASKEDGE   = 0.05     // Required edge for a sell order
)

var (
	api = bitfinex.New(os.Getenv("BITFINEX_KEY"), os.Getenv("BITFINEX_SECRET"))
)

func main() {
	fmt.Println("\nConnecting...")

	// Create channels
	bookChan := make(chan bitfinex.Book)
	tradesChan := make(chan bitfinex.Trades)
	bidChan := make(chan bitfinex.Order)
	askChan := make(chan bitfinex.Order)
	inputChan := make(chan rune)

	// Initial orders
	bid, ask := createOrders()

	// Check for input to break loop
	go checkStdin(inputChan)

	var (
		trades bitfinex.Trades
		book   bitfinex.Book
		start  time.Time
		theo   float64
	)

loop:
	for {
		start = time.Now()

		// Get data in separate goroutines
		go processBook(bookChan)
		go processTrades(tradesChan)

		// Modify orders in separate goroutines when trade data returns
		trades = <-tradesChan
		theo = calculateTheo(trades)
		go replaceBid(bid, bidChan, theo)
		go replaceAsk(ask, askChan, theo)

		// Print data and current orders when all communication is finished
		bid = <-bidChan
		ask = <-askChan
		book = <-bookChan
		printResults(book, trades, bid, ask)

		// Print processing time
		fmt.Printf("\n%v processing time...", time.Since(start))

		// Exit if anything entered by user
		select {
		case <-inputChan:
			cancelAll()
			break loop
		default:
		}
	}
}

// Create initial orders
func createOrders() (bitfinex.Order, bitfinex.Order) {
	// Get the current price
	trades, err := api.Trades(SYMBOL, 1)
	checkErr(err)
	price := trades[0].Price

	// Order parameters
	params := []bitfinex.OrderParams{
		{SYMBOL, AMOUNT, price - BIDEDGE, "bitfinex", "buy", "limit"},
		{SYMBOL, AMOUNT, price + ASKEDGE, "bitfinex", "sell", "limit"},
	}

	// Send new order request to the exchange
	orders, err := api.MultipleNewOrders(params)
	checkErr(err)

	return orders.Orders[0], orders.Orders[1]
}

func checkStdin(inputChan chan rune) {
	var ch rune
	fmt.Scanf("%c", &ch)
	inputChan <- ch
}

// Get book data and send to channel
func processBook(bookChan chan<- bitfinex.Book) {
	book, err := api.Orderbook(SYMBOL, 5, 5)
	checkErr(err)

	bookChan <- book
}

// Get trade data and send to channel
func processTrades(tradesChan chan<- bitfinex.Trades) {
	trades, err := api.Trades(SYMBOL, TRADENUM)
	checkErr(err)

	tradesChan <- trades
}

// Calculate a volume-weighted moving average of trades
func calculateTheo(trades bitfinex.Trades) float64 {
	var sum1, sum2 float64
	for _, trade := range trades {
		sum1 += trade.Price * trade.Amount
		sum2 += trade.Amount
	}
	return sum1 / sum2
}

// Modify bid order and send to channel
func replaceBid(bid bitfinex.Order, bidChan chan<- bitfinex.Order, theo float64) {
	price := theo - BIDEDGE

	var err error
	if math.Abs(price-bid.Price) >= MINCHANGE {
		bid, err = api.ReplaceOrder(bid.ID, SYMBOL, AMOUNT, price, "bitfinex", "buy", "limit")
		checkErr(err)
		if bid.ID == 0 {
			cancelAll()
			log.Fatal("Failed to replace bid order")
		}
	}

	bidChan <- bid
}

// Modify ask order and send to channel
func replaceAsk(ask bitfinex.Order, askChan chan<- bitfinex.Order, theo float64) {
	price := theo + ASKEDGE

	var err error
	if math.Abs(price-ask.Price) >= MINCHANGE {
		ask, err = api.ReplaceOrder(ask.ID, SYMBOL, AMOUNT, price, "bitfinex", "sell", "limit")
		checkErr(err)
		if ask.ID == 0 {
			cancelAll()
			log.Fatal("Failed to replace ask order")
		}
	}

	askChan <- ask
}

// Print results
func printResults(book bitfinex.Book, trades bitfinex.Trades, bid, ask bitfinex.Order) {
	clearScreen()

	fmt.Println("----------------------------")
	fmt.Printf("%-10s%-10s%8s\n", " Bid", "  Ask", "Size ")
	fmt.Println("----------------------------")
	for i := range book.Asks {
		item := book.Asks[len(book.Asks)-1-i]
		fmt.Printf("%-10s%-10.4f%8.2f\n", "", item.Price, item.Amount)
	}
	for _, item := range book.Bids {
		fmt.Printf("%-10.4f%-10.2s%8.2f\n", item.Price, "", item.Amount)
	}
	fmt.Println("----------------------------")

	fmt.Println("\nLast Trades:")
	for _, trade := range trades {
		fmt.Printf("%-6.4f - size: %6.2f\n", trade.Price, trade.Amount)
	}

	fmt.Printf("\nCurrent Bid: %6.4f\n", bid.Price)
	fmt.Printf("Current Ask: %6.4f\n", ask.Price)
}

// Exit on any errors
func checkErr(err error) {
	if err != nil {
		cancelAll()
		log.Fatal(err)
	}
}

// Clear the terminal between prints
func clearScreen() {
	c := exec.Command("clear")
	c.Stdout = os.Stdout
	c.Run()
}

// Cancel all orders
func cancelAll() {
	cancelled := false
	for !cancelled {
		cancelled, _ = api.CancelAll()
	}
	fmt.Println("\nALL ORDERS HAVE BEEN CANCELLED")
}
