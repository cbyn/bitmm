// Simple bitcoin market-making program for bitfinex exchange

package main

import (
	"./bitfinex"
	"fmt"
	"os"
	"os/exec"
	"time"
)

var APIKey = os.Getenv("BITFINEX_API_KEY")
var APISecret = os.Getenv("BITFINEX_API_SECRET")

var apiPublic = bitfinex.New("", "")
var apiPrivate = bitfinex.New(APIKey, APISecret)

func main() {
	for {
		processBook()
	}
}

// Get and print order book data
func processBook() {
	defer timeTrack(time.Now())
	book := apiPublic.GetBook("btcusd?limit_bids=10&limit_asks=10")
	clearScreen()
	printBook(book)
}

// Used to time the processBook function call
func timeTrack(start time.Time) {
	elapsed := time.Since(start)
	fmt.Printf("\n%s to retrieve data\n", elapsed)
}

// Clear the terminal between prints
func clearScreen() {
	c := exec.Command("clear")
	c.Stdout = os.Stdout
	c.Run()
}

// Print the book data
func printBook(book bitfinex.Orderbook) {
	fmt.Printf("%-10s%-10s%10s\n", "Bid", "Ask", "Size")
	fmt.Println("------------------------------")
	for i := range book.Asks {
		item := book.Asks[len(book.Asks)-1-i]
		fmt.Printf("%-10s%-10.2f%10.2f\n", "", item.Price, item.Amount)
	}
	for _, item := range book.Bids {
		fmt.Printf("%-10.2f%-10.2s%10.2f\n", item.Price, "", item.Amount)
	}
}
