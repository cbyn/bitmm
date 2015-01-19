package main

import (
	"bitmm/bitfinex"
	"fmt"
	// "github.com/davecgh/go-spew/spew"
	"log"
	"math"
	"os"
	"os/exec"
	"time"
)

// Trade inputs
const (
	SYMBOL    = "ltcusd" // Instrument to trade
	MINCHANGE = 0.00005  // Minumum change required to update prices
	TRADENUM  = 10       // Number of trades to use in calculations
	AMOUNT    = 0.10     // Size to trade
	BIDEDGE   = 0.005    // Required edge for a buy order
	ASKEDGE   = 0.005    // Required edge for a sell order
)

var api = bitfinex.New(os.Getenv("BITFINEX_KEY"), os.Getenv("BITFINEX_SECRET"))

func main() {
	fmt.Println("\nInitializing...")

	// Check for input to break loop
	inputChan := make(chan rune)
	go checkStdin(inputChan)

	// Run loop until user input is received
	runMainLoop(inputChan)
}

// Check for any user input
func checkStdin(inputChan chan<- rune) {
	var ch rune
	fmt.Scanf("%c", &ch)
	inputChan <- ch
}

// Infinite loop
func runMainLoop(inputChan <-chan rune) {
	// Exchange communication channels
	bookChan := make(chan bitfinex.Book)
	tradesChan := make(chan bitfinex.Trades)
	ordersChan := make(chan bitfinex.Orders)
	positionChan := make(chan float64)

	var (
		trades      bitfinex.Trades
		book        bitfinex.Book
		orders      bitfinex.Orders
		start       time.Time
		oldPosition float64
		newPosition float64
		oldTheo     float64
		newTheo     float64
	)

	for {
		// Record time for each iteration
		start = time.Now()

		// Get data in separate goroutines
		go processTrades(tradesChan)
		go processBook(bookChan)
		go checkPosition(positionChan)

		// Send orders when trades and position data returns
		trades = <-tradesChan
		newTheo = calculateTheo(trades)
		newPosition = <-positionChan
		go sendOrders(orders, oldTheo, newTheo, oldPosition, newPosition, ordersChan)

		oldTheo = newTheo
		oldPosition = newPosition

		// Print results when book and order data returns
		book = <-bookChan
		orders = <-ordersChan
		printResults(book, trades, orders, newTheo, newPosition, start)

		// Exit if anything entered by user
		select {
		case <-inputChan:
			exit()
			return
		default:
		}
	}
}

// Send orders to the exchange
func sendOrders(orders bitfinex.Orders, oldTheo, newTheo, oldPosition,
	newPosition float64, ordersChan chan<- bitfinex.Orders) {

	if math.Abs(oldTheo-newTheo) > MINCHANGE || math.Abs(oldPosition-
		newPosition) > 0.01 {
		// First cancel all orders
		cancelAll()

		var params []bitfinex.OrderParams

		if newPosition+AMOUNT < 0.01 { // Max short postion
			// One order at value to exit position
			params = []bitfinex.OrderParams{
				{SYMBOL, -newPosition, newTheo, "bitfinex", "buy", "limit"},
			}
		} else if newPosition-AMOUNT > -0.01 { // Max long postion
			// One order at value to exit position
			params = []bitfinex.OrderParams{
				{SYMBOL, newPosition, newTheo, "bitfinex", "sell", "limit"},
			}
		} else {
			// Two orders for edge
			params = []bitfinex.OrderParams{
				{SYMBOL, AMOUNT - newPosition, newTheo - BIDEDGE, "bitfinex", "buy", "limit"},
				{SYMBOL, AMOUNT + newPosition, newTheo + ASKEDGE, "bitfinex", "sell", "limit"},
			}
		}

		// Send new order request to the exchange
		orders, err := api.MultipleNewOrders(params)
		checkErr(err)
		ordersChan <- orders
	} else {
		ordersChan <- orders
	}
}

func checkPosition(positionChan chan<- float64) {
	position := 0.0
	posSlice, err := api.ActivePositions()
	checkErr(err)
	for _, pos := range posSlice {
		if pos.Symbol == SYMBOL {
			position = pos.Amount
		}
	}

	positionChan <- position
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

// Called on any error
func checkErr(err error) {
	if err != nil {
		exit()
		log.Fatal(err)
	}
}

// Call on exit
func exit() {
	cancelAll()
	fmt.Println("\nCancelled all orders.")
}

// Cancel all orders
func cancelAll() {
	cancelled := false
	for !cancelled {
		cancelled, _ = api.CancelAll()
	}
}

// Print results
func printResults(book bitfinex.Book, trades bitfinex.Trades,
	orders bitfinex.Orders, theo, position float64, start time.Time) {

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

	fmt.Printf("\nPosition: %.2f\n", position)
	fmt.Printf("Theo:     %.4f\n", theo)

	fmt.Println("\nActive orders:")
	for _, order := range orders.Orders {
		fmt.Printf("%6.2f %s @ %6.4f\n", order.Amount, SYMBOL, order.Price)
	}

	fmt.Printf("\n%v processing time...", time.Since(start))
}

// Clear the terminal between prints
func clearScreen() {
	c := exec.Command("clear")
	c.Stdout = os.Stdout
	c.Run()
}
