// TODO: set a global orders theo for comparisons

package main

import (
	"bitmm/bitfinex"
	"fmt"
	"math"
	"os"
	"os/exec"
	"time"
)

// Trade inputs
const (
	SYMBOL    = "btcusd" // Instrument to trade
	MINCHANGE = 0.01     // Minumum change required to update prices
	TRADENUM  = 10       // Number of trades to use in calculations
	AMOUNT    = 0.01     // Size to trade
	BIDEDGE   = 0.10     // Required edge for a buy order
	ASKEDGE   = 0.10     // Required edge for a sell order
)

var (
	api        = bitfinex.New(os.Getenv("BITFINEX_KEY"), os.Getenv("BITFINEX_SECRET"))
	apiErrors  = false
	liveOrders = false
	orderTheo  = 0.0
)

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

	var (
		trades      bitfinex.Trades
		book        bitfinex.Book
		orders      bitfinex.Orders
		start       time.Time
		oldPosition float64
		newPosition float64
		theo        float64
		lastTrade   int
	)

	for {
		// Record time for each iteration
		start = time.Now()

		// Get data in separate goroutines
		go processTrades(tradesChan)
		go processBook(bookChan)

		// Possibly send orders when trades data returns
		trades = <-tradesChan
		if !apiErrors && trades[0].Timestamp != lastTrade { // If new trades
			theo = calculateTheo(trades)
			newPosition = checkPosition()
			// Reset for next iteration
			lastTrade = trades[0].Timestamp
		}
		go sendOrders(orders, theo, oldPosition, newPosition, ordersChan)

		// Print results when book and order data returns
		book = <-bookChan
		orders = <-ordersChan
		if !apiErrors {
			printResults(book, trades, orders, theo, newPosition, start)
		}

		// Exit if anything entered by user
		select {
		case <-inputChan:
			exit()
			return
		default:
		}

		// Reset for next iteration
		apiErrors = false
		oldPosition = newPosition
	}
}

// Send orders to the exchange
func sendOrders(orders bitfinex.Orders, theo, oldPosition, newPosition float64,
	ordersChan chan<- bitfinex.Orders) {

	if (math.Abs(theo-orderTheo) > MINCHANGE || math.Abs(oldPosition-
		newPosition) > 0.01 || !liveOrders) && !apiErrors {

		orderTheo = theo

		if liveOrders {
			cancelAll()
		}

		var params []bitfinex.OrderParams

		if newPosition+AMOUNT < 0.01 { // Max short postion
			// One order at theo to exit position
			params = []bitfinex.OrderParams{
				{SYMBOL, -newPosition, theo - BIDEDGE, "bitfinex", "buy", "limit"},
			}
		} else if newPosition-AMOUNT > -0.01 { // Max long postion
			// One order at theo to exit position
			params = []bitfinex.OrderParams{
				{SYMBOL, newPosition, theo + ASKEDGE, "bitfinex", "sell", "limit"},
			}
		} else {
			// Two orders for edge
			params = []bitfinex.OrderParams{
				{SYMBOL, math.Min(AMOUNT-newPosition, AMOUNT), theo - BIDEDGE,
					"bitfinex", "buy", "limit"},
				{SYMBOL, math.Min(AMOUNT+newPosition, AMOUNT), theo + ASKEDGE,
					"bitfinex", "sell", "limit"},
			}
		}

		// Send new order request to the exchange
		orders, err := api.MultipleNewOrders(params)
		liveOrders = true
		checkErr(err)
		ordersChan <- orders
	} else {
		ordersChan <- orders
	}
}

func checkPosition() float64 {
	var position float64
	posSlice, err := api.ActivePositions()
	checkErr(err)
	for _, pos := range posSlice {
		if pos.Symbol == SYMBOL {
			position = pos.Amount
		}
	}

	return position
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
		cancelAll()
		apiErrors = true
		fmt.Println(err)
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
	liveOrders = false
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
		fmt.Printf("%8.2f %s @ %6.4f\n", order.Amount, SYMBOL, order.Price)
	}

	fmt.Printf("\n%v processing time...", time.Since(start))
}

// Clear the terminal between prints
func clearScreen() {
	c := exec.Command("clear")
	c.Stdout = os.Stdout
	c.Run()
}
