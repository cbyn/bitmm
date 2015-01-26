package main

import (
	"bitmm/bitfinex"
	"fmt"
	"log"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"
)

// Trade inputs
const (
	SYMBOL    = "ltcusd" // Instrument to trade
	MINCHANGE = 0.0025   // Minumum change required to update prices
	TRADENUM  = 20       // Number of trades to use in calculations
	MINO      = 1        // Min order size
)

var (
	// MAXO maximum order size
	MAXO float64
	// INEDGE edge for position entry orders
	INEDGE float64
	// OUTEDGE edge for position exit orders
	OUTEDGE    float64
	api        = bitfinex.New(os.Getenv("BITFINEX_KEY"), os.Getenv("BITFINEX_SECRET"))
	apiErrors  = false
	liveOrders = false
	orderTheo  = 0.0 // Theo value on which the live orders are based
	orderPos   = 0.0 // Position on which the live orders are based
)

func main() {
	if len(os.Args) < 4 {
		fmt.Printf("usage: %s <size> <entry-edge> <exit-edge>\n", filepath.Base(os.Args[0]))
		os.Exit(1)
	}

	fmt.Println("\nInitializing...")

	// Get MAXO, INEDGE, OUTEDGE from user input
	getVars()

	// Check for input to break loop
	inputChan := make(chan rune)
	go checkStdin(inputChan)

	// Run loop until user input is received
	runMainLoop(inputChan)
}

func getVars() {
	var err error
	if MAXO, err = strconv.ParseFloat(os.Args[1], 64); err != nil {
		log.Fatal(err)
	}
	if INEDGE, err = strconv.ParseFloat(os.Args[2], 64); err != nil {
		log.Fatal(err)
	}
	if OUTEDGE, err = strconv.ParseFloat(os.Args[3], 64); err != nil {
		log.Fatal(err)
	}
}

// Check for any user input
func checkStdin(inputChan chan<- rune) {
	var ch rune
	fmt.Scanf("%c", &ch)
	inputChan <- ch
}

// Infinite loop
func runMainLoop(inputChan <-chan rune) {

	var (
		trades bitfinex.Trades
		// book        bitfinex.Book
		orders    bitfinex.Orders
		start     time.Time
		position  float64
		theo      float64
		lastTrade int
	)

	for {
		// Record time for each iteration
		start = time.Now()

		// Exit if anything entered by user
		select {
		case <-inputChan:
			exit()
			return
		default:
		}

		trades = processTrades()
		if !apiErrors && trades[0].TID != lastTrade { // If new trades
			theo = calculateTheo(trades)
			position = checkPosition()
			if (math.Abs(theo-orderTheo) >= MINCHANGE || math.Abs(position-
				orderPos) >= MINO || !liveOrders) && !apiErrors {
				orders = sendOrders(theo, position)
			}
		}

		if !apiErrors {
			printResults(trades, orders, theo, position, start)
			lastTrade = trades[0].TID
		}

		// Reset for next iteration
		apiErrors = false
	}
}

// Send orders to the exchange
func sendOrders(theo, position float64) bitfinex.Orders {
	if liveOrders {
		cancelAll()
	}

	// Send new order request to the exchange
	params := calcOrderParams(position, theo)
	orders, err := api.MultipleNewOrders(params)
	liveOrders = true
	checkErr(err)
	if err == nil {
		orderTheo = theo
		orderPos = position
	}
	return orders
}

func calcOrderParams(position, theo float64) []bitfinex.OrderParams {
	var params []bitfinex.OrderParams

	if math.Abs(position) < MINO { // No position
		params = []bitfinex.OrderParams{
			{SYMBOL, MAXO, theo - INEDGE, "bitfinex", "buy", "limit"},
			{SYMBOL, MAXO, theo + INEDGE, "bitfinex", "sell", "limit"},
		}
	} else if position < (-1*MAXO)+MINO { // Max short postion
		params = []bitfinex.OrderParams{
			{SYMBOL, -1 * position, theo - OUTEDGE, "bitfinex", "buy", "limit"},
		}
	} else if position > MAXO-MINO { // Max long postion
		params = []bitfinex.OrderParams{
			{SYMBOL, position, theo + OUTEDGE, "bitfinex", "sell", "limit"},
		}
	} else if (-1*MAXO)+MINO <= position && position <= -1*MINO { // Partial short
		params = []bitfinex.OrderParams{
			{SYMBOL, MAXO, theo - INEDGE, "bitfinex", "buy", "limit"},
			{SYMBOL, -1 * position, theo - OUTEDGE, "bitfinex", "buy", "limit"},
			{SYMBOL, MAXO + position, theo + INEDGE, "bitfinex", "sell", "limit"},
		}
	} else if MINO <= position && position <= MAXO-MINO { // Partial long
		params = []bitfinex.OrderParams{
			{SYMBOL, MAXO - position, theo - INEDGE, "bitfinex", "buy", "limit"},
			{SYMBOL, position, theo + OUTEDGE, "bitfinex", "sell", "limit"},
			{SYMBOL, MAXO, theo + INEDGE, "bitfinex", "sell", "limit"},
		}
	}

	return params
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

// Get trade data
func processTrades() bitfinex.Trades {
	trades, err := api.Trades(SYMBOL, TRADENUM)
	checkErr(err)

	return trades
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
func printResults(trades bitfinex.Trades,
	orders bitfinex.Orders, theo, position float64, start time.Time) {

	clearScreen()

	fmt.Println("\nLast Trades:")
	for i := 0; i < 10; i++ {
		fmt.Printf("%-6.4f - size: %6.2f\n", trades[i].Price, trades[i].Amount)
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
