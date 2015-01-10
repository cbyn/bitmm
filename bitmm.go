// Simple bitcoin market-making program for bitfinex exchange

package main

import (
	"./bitfinex"
	"fmt"
	"os"
	"os/exec"
	"time"
)

func main() {
	b := bitfinex.Bitfinex{
		"https://api.bitfinex.com/v1/book/btcusd?limit_bids=25&limit_asks=25",
	}
	for {
		processData(b)
	}
}

// Get and print data
func processData(b bitfinex.Bitfinex) {
	defer timeTrack(time.Now())
	book := b.GetBook()
	clearScreen()
	printBook(book)
}

// Used to time the processData func call
func timeTrack(start time.Time) {
	elapsed := time.Since(start)
	fmt.Printf("\n%s to retrieve data\n", elapsed)
}

// Clear the terminal
func clearScreen() {
	c := exec.Command("clear")
	c.Stdout = os.Stdout
	c.Run()
}

// Print the book data
func printBook(book bitfinex.Book) {
	fmt.Printf("Bid\tAsk\tVol\n")
	fmt.Println("--------------------------")
	for i := range book.Asks {
		item := book.Asks[len(book.Asks)-1-i]
		fmt.Printf("\t%s\t%s\n", item.Price, item.Amount)
	}
	for _, item := range book.Bids {
		fmt.Printf("%s\t\t%s\n", item.Price, item.Amount)
	}
}
