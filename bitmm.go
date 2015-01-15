// This only shows a streaming orderbook right now

package main

import (
	"./bitfinex"
	"fmt"
	"os"
	"os/exec"
	"time"
)

var (
	api = bitfinex.New(os.Getenv("BITFINEX_KEY"), os.Getenv("BITFINEX_SECRET"))
)

func main() {
	bookChan := make(chan bitfinex.Book)
	tradesChan := make(chan bitfinex.Trades)

	for {
		start := time.Now()

		// get data in separate goroutines
		go processBook(bookChan)
		go processTrades(tradesChan)

		// block until both are done
		printResults(<-bookChan, <-tradesChan)

		fmt.Printf("\n%v to get data\n", time.Since(start))
	}
}

// Get book data and send to channel
func processBook(bookChan chan<- bitfinex.Book) {
	book, _ := api.Orderbook("btcusd", 5, 5)
	bookChan <- book
}

// Get trade data and send to channel
func processTrades(tradesChan chan<- bitfinex.Trades) {
	trades, _ := api.Trades("btcusd", 10)
	tradesChan <- trades
}

// Print results
func printResults(book bitfinex.Book, trades bitfinex.Trades) {
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
}

// Clear the terminal between prints
func clearScreen() {
	c := exec.Command("clear")
	c.Stdout = os.Stdout
	c.Run()
}
