// This only shows a streaming orderbook right now

package main

import (
	"./bitfinex"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"
)

var (
	APIKey     = os.Getenv("BITFINEX_KEY")
	APISecret  = os.Getenv("BITFINEX_SECRET")
	apiPublic  = bitfinex.New("", "")
	apiPrivate = bitfinex.New(APIKey, APISecret)
)

func main() {
	for {
		processBook()
	}
}

// Get and print order book data
func processBook() {
	defer timeTrack(time.Now())
	book, err := apiPublic.Orderbook("btcusd", 10, 10)
	if err != nil {
		log.Fatal(err)
	}
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
func printBook(book bitfinex.Book) {
	fmt.Printf("%-10s%-10s\n", " Bid", " Ask")
	fmt.Println("--------------------------")
	for i := range book.Asks {
		item := book.Asks[len(book.Asks)-1-i]
		fmt.Printf("%-10s%-10.2f%6.2f\n", "", item.Price, item.Amount)
	}
	for _, item := range book.Bids {
		fmt.Printf("%-10.2f%-10.2s%6.2f\n", item.Price, "", item.Amount)
	}
}
