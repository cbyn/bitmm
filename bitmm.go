package main

import (
	"bitmm/bitfinex"
	"fmt"
	"github.com/grd/stat"
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
	SYMBOL    = "btcusd" // Instrument to trade
	MINCHANGE = 0.1      // Minumum change required to update prices
	TRADENUM  = 40       // Number of trades to use in calculations
	MINO      = 0.01     // Min order size
)

var (
	api        = bitfinex.New(os.Getenv("BITFINEX_KEY"), os.Getenv("BITFINEX_SECRET"))
	apiErrors  = false // Set to true on any error
	liveOrders = false // Set to true on any order
	orderTheo  = 0.0   // Theo value on which the live orders are based
	orderPos   = 0.0   // Position on which the live orders are based
	// Fed in as OS args:
	maxPos      float64 // Maximum Position size
	minEdge     float64 // Minimum edge for position entry
	stdMult     float64 // Multiplier for standard deviation
	exitPercent float64 // Percent of edge for position exit
)

func main() {
	if len(os.Args) < 5 {
		fmt.Printf("usage: %s <size> <minimum edge> <stdev multiplier> <exit percent edge>\n",
			filepath.Base(os.Args[0]))
		os.Exit(1)
	}

	fmt.Println("\nInitializing...")

	// Set file for logging
	logFile, err := os.OpenFile("bitmm_log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal(err)
	}
	defer logFile.Close()
	log.SetOutput(logFile)

	// Get maxPos, minEdge, exitPercent from user input
	getArgs()

	// Check for input to break loop
	inputChan := make(chan rune)
	go checkStdin(inputChan)

	// Run loop until user input is received
	runMainLoop(inputChan)
}

func getArgs() {
	var err error
	if maxPos, err = strconv.ParseFloat(os.Args[1], 64); err != nil {
		log.Fatal(err)
	}
	if minEdge, err = strconv.ParseFloat(os.Args[2], 64); err != nil {
		log.Fatal(err)
	}
	if stdMult, err = strconv.ParseFloat(os.Args[3], 64); err != nil {
		log.Fatal(err)
	}
	if exitPercent, err = strconv.ParseFloat(os.Args[4], 64); err != nil {
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
	positionChan := make(chan float64)

	var (
		trades    bitfinex.Trades
		orders    bitfinex.Orders
		start     time.Time
		position  float64
		theo      float64
		stdev     float64
		lastTrade int
	)

	for {
		// Record time for each iteration
		start = time.Now()

		// Cancel orders and exit if anything entered by user
		select {
		case <-inputChan:
			exit()
			return
		default: // Continue if nothing on chan
		}

		trades = getTrades()
		if !apiErrors && trades[0].TID != lastTrade { // If new trades
			go checkPosition(positionChan)
			// Do calcs on trade data while waiting for position data
			theo = calculateTheo(trades)
			stdev = calculateStdev(trades)
			position = <-positionChan
			if (math.Abs(theo-orderTheo) >= MINCHANGE || math.Abs(position-
				orderPos) >= MINO || !liveOrders) && !apiErrors {
				orders = sendOrders(theo, position, stdev)
			}
		}

		if !apiErrors {
			printResults(orders, position, stdev, theo, start)
			// Reset for next iteration
			lastTrade = trades[0].TID
		}

		// Reset for next iteration
		apiErrors = false
	}
}

// Send orders to the exchange
func sendOrders(theo, position, stdev float64) bitfinex.Orders {
	if liveOrders {
		cancelAll()
	}
	liveOrders = true
	orderTheo = theo
	orderPos = position

	// Send new order request to the exchange
	params := calculateOrderParams(position, theo, stdev)
	orders, err := api.MultipleNewOrders(params)
	checkErr(err, "MultipleNewOrders")
	if orders.Message != "" {
		cancelAll()
		log.Printf("Order Message: %s\n", orders.Message)
	}
	return orders
}

func calculateOrderParams(position, theo, stdev float64) []bitfinex.OrderParams {
	var params []bitfinex.OrderParams

	if math.Abs(position) < MINO { // No position
		params = []bitfinex.OrderParams{
			{SYMBOL, maxPos, theo - math.Max(stdev, minEdge), "bitfinex", "buy", "limit"},
			{SYMBOL, maxPos, theo + math.Max(stdev, minEdge), "bitfinex", "sell", "limit"},
		}
	} else if position < (-1*maxPos)+MINO { // Max short postion
		params = []bitfinex.OrderParams{
			{SYMBOL, -1 * position, theo - math.Max(stdev, minEdge)*exitPercent, "bitfinex", "buy", "limit"},
		}
	} else if position > maxPos-MINO { // Max long postion
		params = []bitfinex.OrderParams{
			{SYMBOL, position, theo + math.Max(stdev, minEdge)*exitPercent, "bitfinex", "sell", "limit"},
		}
	} else if (-1*maxPos)+MINO <= position && position <= -1*MINO { // Partial short
		params = []bitfinex.OrderParams{
			{SYMBOL, maxPos, theo - math.Max(stdev, minEdge), "bitfinex", "buy", "limit"},
			{SYMBOL, -1 * position, theo - math.Max(stdev, minEdge)*exitPercent, "bitfinex", "buy", "limit"},
			{SYMBOL, maxPos + position, theo + math.Max(stdev, minEdge), "bitfinex", "sell", "limit"},
		}
	} else if MINO <= position && position <= maxPos-MINO { // Partial long
		params = []bitfinex.OrderParams{
			{SYMBOL, maxPos - position, theo - math.Max(stdev, minEdge), "bitfinex", "buy", "limit"},
			{SYMBOL, position, theo + math.Max(stdev, minEdge)*exitPercent, "bitfinex", "sell", "limit"},
			{SYMBOL, maxPos, theo + math.Max(stdev, minEdge), "bitfinex", "sell", "limit"},
		}
	}

	return params
}

func checkPosition(positionChan chan<- float64) {
	var position float64
	posSlice, err := api.ActivePositions()
	checkErr(err, "ActivePositions")
	for _, pos := range posSlice {
		if pos.Symbol == SYMBOL {
			position = pos.Amount
		}
	}

	positionChan <- position
}

// Get trade data
func getTrades() bitfinex.Trades {
	trades, err := api.Trades(SYMBOL, TRADENUM)
	checkErr(err, "Trades")

	return trades
}

// Calculate a volume and time weighted average of traded prices
func calculateTheo(trades bitfinex.Trades) float64 {
	weightDuration := 60 // number of seconds back for a 50% weight relative to most recent
	mostRecent := trades[0].Timestamp
	var weight, timeDivisor, sum, weightTotal float64

	for _, trade := range trades {
		timeDivisor = float64(mostRecent - trade.Timestamp + weightDuration)
		weight = trade.Amount / timeDivisor
		sum += trade.Price * weight
		weightTotal += weight
	}

	return sum / weightTotal
}

func calculateStdev(trades bitfinex.Trades) float64 {
	x := make(stat.Float64Slice, len(trades)-1)
	for i := 1; i < len(trades); i++ {
		x[i-1] = trades[i-1].Price - trades[i].Price
	}
	return stdMult * stat.Sd(x)
}

// Called on any error
func checkErr(err error, methodName string) {
	if err != nil {
		cancelAll()
		log.Printf("%s Error: %s\n", methodName, err)
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
func printResults(orders bitfinex.Orders, position, stdev, theo float64, start time.Time) {

	clearScreen()

	fmt.Printf("\nPosition: %.2f\n", position)
	fmt.Printf("Stdev:    %.4f\n", stdev)
	fmt.Printf("Theo:     %.4f\n", theo)

	fmt.Println("\nActive orders:")
	for _, order := range orders.Orders {
		fmt.Printf("%7.2f %s @ %6.4f\n", order.Amount, SYMBOL, order.Price)
	}

	fmt.Printf("\n%v processing time...", time.Since(start))
}

// Clear the terminal between prints
func clearScreen() {
	c := exec.Command("clear")
	c.Stdout = os.Stdout
	c.Run()
}
