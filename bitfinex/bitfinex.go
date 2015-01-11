// Package to communicate with bitfinex exchange

package bitfinex

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
)

const (
	APIURL = "https://api.bitfinex.com/v1/" // Bitfinex API URL
)

// Stores Bitfinex API credentials
type API struct {
	APIKey    string
	APISecret string
}

// Order book data from the exchange
type Orderbook struct {
	Bids []OrderbookItems // Slice of bid data items
	Asks []OrderbookItems // Slice of ask data items
}

// Inner order book data from the exchange
type OrderbookItems struct {
	Price     float64 `json:"price,string"`     // Order price
	Amount    float64 `json:"amount,string"`    // Order volume
	Timestamp float64 `json:"timestamp,string"` // Exchange timestamp
}

// Return a new Bitfinex API instance
func New(key, secret string) (api *API) {
	api = &API{
		APIKey:    key,
		APISecret: secret,
	}
	return api
}

// Get book data from exchange
func (api *API) GetBook(url string) (book Orderbook) {
	data, err := api.get("book/" + url)
	if err != nil {
		log.Fatal(err)
	}
	err = json.Unmarshal(data, &book)
	if err != nil {
		log.Fatal(err)
	}
	return
}

// API GET
func (api *API) get(url string) (body []byte, err error) {
	resp, err := http.Get(APIURL + url)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	body, err = ioutil.ReadAll(resp.Body)
	return
}
