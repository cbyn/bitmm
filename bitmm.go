package main

import (
	"bitmm/bitfinex"
	"code.google.com/p/gcfg"
	"flag"
	"fmt"
	"github.com/grd/stat"
	"log"
	"math"
	"os"
	"os/exec"
	"time"
)

// Config stores user configuration
type Config struct {
	Sec struct {
		Symbol         string  // Instrument to trade
		TradeNum       int     // Number of trades to use in calculations
		WeightDuration int     // Number of seconds back for a 50% weight
		MinPos         float64 // Min order size
		MaxPos         float64 // Maximum Position size
		MinEdge        float64 // Minimum edge for position entry
		StdMult        float64 // Multiplier for standard deviation
		ExitPercent    float64 // Percent of edge for position exit
		MinChange      float64 // Minumum change required to update prices
	}
}

var (
	api        = bitfinex.New(os.Getenv("BITFINEX_KEY"), os.Getenv("BITFINEX_SECRET"))
	apiErrors  = false // Set to true on any error
	liveOrders = false // Set to true on any order
	orderTheo  = 0.0   // Theo value on which the live orders are based
	orderPos   = 0.0   // Position on which the live orders are based
	cfg        Config
)

func main() {
	fmt.Println("\nInitializing...")

	// Set file for logging
	logFile, err := os.OpenFile("bitmm.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal(err)
	}
	defer logFile.Close()
	log.SetOutput(logFile)

	// Get config info
	configFile := flag.String("config", "bitmm.gcfg", "Configuration file")
	flag.Parse()
	err = gcfg.ReadFileInto(&cfg, *configFile)
	if err != nil {
		log.Fatal(err)
	}

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
			if (math.Abs(theo-orderTheo) >= cfg.Sec.MinChange || math.Abs(position-
				orderPos) >= cfg.Sec.MinPos || !liveOrders) && !apiErrors {
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

	if orders.Message != "" || len(orders.Orders) == 0 || orders.Orders[0].ID == 0 {
		cancelAll()
		log.Printf("Order Problem %s\n", orders.Message)
	}

	return orders
}

func calculateOrderParams(position, theo, stdev float64) []bitfinex.OrderParams {
	var params []bitfinex.OrderParams

	if math.Abs(position) < cfg.Sec.MinPos { // No position
		params = []bitfinex.OrderParams{
			{cfg.Sec.Symbol, cfg.Sec.MaxPos, theo - math.Max(stdev, cfg.Sec.MinEdge), "bitfinex", "buy", "limit"},
			{cfg.Sec.Symbol, cfg.Sec.MaxPos, theo + math.Max(stdev, cfg.Sec.MinEdge), "bitfinex", "sell", "limit"},
		}
	} else if position < (-1*cfg.Sec.MaxPos)+cfg.Sec.MinPos { // Max short postion
		params = []bitfinex.OrderParams{
			{cfg.Sec.Symbol, -1 * position, theo - math.Max(stdev, cfg.Sec.MinEdge)*cfg.Sec.ExitPercent, "bitfinex", "buy", "limit"},
		}
	} else if position > cfg.Sec.MaxPos-cfg.Sec.MinPos { // Max long postion
		params = []bitfinex.OrderParams{
			{cfg.Sec.Symbol, position, theo + math.Max(stdev, cfg.Sec.MinEdge)*cfg.Sec.ExitPercent, "bitfinex", "sell", "limit"},
		}
	} else if (-1*cfg.Sec.MaxPos)+cfg.Sec.MinPos <= position && position <= -1*cfg.Sec.MinPos { // Partial short
		params = []bitfinex.OrderParams{
			{cfg.Sec.Symbol, cfg.Sec.MaxPos, theo - math.Max(stdev, cfg.Sec.MinEdge), "bitfinex", "buy", "limit"},
			{cfg.Sec.Symbol, -1 * position, theo - math.Max(stdev, cfg.Sec.MinEdge)*cfg.Sec.ExitPercent, "bitfinex", "buy", "limit"},
			{cfg.Sec.Symbol, cfg.Sec.MaxPos + position, theo + math.Max(stdev, cfg.Sec.MinEdge), "bitfinex", "sell", "limit"},
		}
	} else if cfg.Sec.MinPos <= position && position <= cfg.Sec.MaxPos-cfg.Sec.MinPos { // Partial long
		params = []bitfinex.OrderParams{
			{cfg.Sec.Symbol, cfg.Sec.MaxPos - position, theo - math.Max(stdev, cfg.Sec.MinEdge), "bitfinex", "buy", "limit"},
			{cfg.Sec.Symbol, position, theo + math.Max(stdev, cfg.Sec.MinEdge)*cfg.Sec.ExitPercent, "bitfinex", "sell", "limit"},
			{cfg.Sec.Symbol, cfg.Sec.MaxPos, theo + math.Max(stdev, cfg.Sec.MinEdge), "bitfinex", "sell", "limit"},
		}
	}

	return params
}

func checkPosition(positionChan chan<- float64) {
	var position float64
	posSlice, err := api.ActivePositions()
	checkErr(err, "ActivePositions")
	for _, pos := range posSlice {
		if pos.Symbol == cfg.Sec.Symbol {
			position = pos.Amount
		}
	}

	positionChan <- position
}

// Get trade data
func getTrades() bitfinex.Trades {
	trades, err := api.Trades(cfg.Sec.Symbol, cfg.Sec.TradeNum)
	checkErr(err, "Trades")

	return trades
}

// Calculate a volume and time weighted average of traded prices
func calculateTheo(trades bitfinex.Trades) float64 {
	mostRecent := trades[0].Timestamp
	var weight, timeDivisor, sum, weightTotal float64

	for _, trade := range trades {
		timeDivisor = float64(mostRecent - trade.Timestamp + cfg.Sec.WeightDuration)
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
	return cfg.Sec.StdMult * stat.Sd(x)
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
		fmt.Printf("%7.2f %s @ %6.4f\n", order.Amount, cfg.Sec.Symbol, order.Price)
	}

	fmt.Printf("\n%v processing time...", time.Since(start))
}

// Clear the terminal between prints
func clearScreen() {
	c := exec.Command("clear")
	c.Stdout = os.Stdout
	c.Run()
}
