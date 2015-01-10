// Package to communicate with bitfinex exchange

package bitfinex

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
)

type Bitfinex struct {
	RequestURL string
}

// JSON data from the exchange
type Book struct {
	Bids []BookItems // Slice of bid data items
	Asks []BookItems // Slice of ask data items
}

// Inner JSON data from the exchange
type BookItems struct {
	Price     string // Order price
	Amount    string // Order volume
	Timestamp string // Exchange timestamp
}

// Method to get book data from exchange
func (b Bitfinex) GetBook() (book Book) {
	data, err := getData(b.RequestURL)
	if err != nil {
		log.Fatal(err)
	}
	err = json.Unmarshal(data, &book)
	if err != nil {
		log.Fatal(err)
	}
	return book
}

// Make an API request
func getData(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return []byte{}, err
	}
	defer resp.Body.Close()
	return ioutil.ReadAll(resp.Body)
}
