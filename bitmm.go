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
	apiPublic = bitfinex.New("", "")
)

func main() {
	for {
		start := time.Now()
		book, bTime, bErr := processBook()
		trades, tTime, tErr := processTrades()
		clearScreen()
		printResults(book, bTime, bErr, trades, tTime, tErr)
		fmt.Printf("%v total\n", time.Since(start))
	}
}

func processBook() (bitfinex.Book, time.Duration, error) {
	start := time.Now()
	trades, err := apiPublic.Orderbook("ltcusd", 5, 5)
	return trades, time.Since(start), err
}

func processTrades() (bitfinex.Trades, time.Duration, error) {
	start := time.Now()
	trades, err := apiPublic.Trades("ltcusd", 5)
	return trades, time.Since(start), err
}

// Print results
func printResults(book bitfinex.Book, bTime time.Duration, bErr error, trades bitfinex.Trades, tTime time.Duration, tErr error) {
	if bErr != nil {
		fmt.Println(bErr)
	} else {
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
	}
	if tErr != nil {
		fmt.Println(tErr)
	} else {
		fmt.Println("\nLast Trades:")
		for _, trade := range trades {
			fmt.Printf("%-6.4f - size: %6.2f\n", trade.Price, trade.Amount)
		}
		fmt.Printf("\n%v to get book data\n", bTime)
		fmt.Printf("%v to get trade data\n", tTime)
	}
}

// Clear the terminal between prints
func clearScreen() {
	c := exec.Command("clear")
	c.Stdout = os.Stdout
	c.Run()
}
