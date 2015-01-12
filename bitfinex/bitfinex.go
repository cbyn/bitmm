// Package to communicate with bitfinex exchange

package bitfinex

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"
)

const (
	APIURL = "https://api.bitfinex.com/" // Bitfinex API URL
)

// Stores Bitfinex API credentials
type API struct {
	APIKey    string
	APISecret string
}

// Error from exchange
type ErrorMessage struct {
	Message string `json:"message"` // Returned only on error
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

type Order struct {
	ID              int     `json:"order_id"`                   // Order ID
	Symbol          string  `json:"symbol"`                     // The symbol name the order belongs to
	Exchange        string  `json:"exchange"`                   // Exchange name "bitfinex"
	Price           float64 `json:"price,string"`               // The price the order was issued at (can be null for market orders)
	ExecutionPrice  float64 `json:"avg_execution_price,string"` // The average price at which this order as been executed so far. 0 if the order has not been executed at all
	Side            string  `json:"side"`                       // Either "buy" or "sell"
	OrderType       string  `json:"type"`                       // Either "market" / "limit" / "stop" / "trailing-stop"
	Timestamp       float64 `json:"timestamp,string"`           // The timestamp the order was submitted
	IsLive          bool    `json:"is_live,bool"`               // Could the order still be filled?
	IsCancelled     bool    `json:"is_cancelled,bool"`          // Has the order been cancelled?
	WasForced       bool    `json:"was_forced,bool"`            // For margin only: true if it was forced by the system
	ExecutedAmount  float64 `json:"executed_amount,string"`     // How much of the order has been executed so far in its history?
	RemainingAmount float64 `json:"remaining_amount,string"`    // How much is still remaining to be submitted?
	OriginalAmount  float64 `json:"original_amount,string"`     // What was the order originally submitted for?
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
func (api *API) GetBook(url string) Orderbook {
	var book Orderbook
	data, err := api.get("/v1/book/" + url)
	if err != nil {
		log.Fatal(err)
	}
	err = json.Unmarshal(data, &book)
	if err != nil {
		log.Fatal(err)
	}

	return book
}

// Post new order

// Request
// symbol (string): The name of the symbol (see `/symbols`).
// amount (decimal): Order size: how much to buy or sell.
// price (price): Price to buy or sell at. Must be positive. Use random number for market orders.
// exchange (string): "bitfinex".
// side (string): Either "buy" or "sell".
// type (string): Either "market" / "limit" / "stop" / "trailing-stop" / "fill-or-kill" / "exchange market" / "exchange limit" / "exchange stop" / "exchange trailing-stop" / "exchange fill-or-kill". (type starting by "exchange " are exchange orders, others are margin trading orders)
// is_hidden (bool) true if the order should be hidden. Default is false.

// Response
// order_id (int): A randomly generated ID for the order.
// and the information given by /order/status
func (api *API) NewOrder(symbol string, amount, price float64, exchange, side, otype string, hidden bool) (Order, error) {
	var order Order
	request := struct {
		URL      string  `json:"request"`
		Nonce    string  `json:"nonce"`
		Symbol   string  `json:"symbol"`
		Amount   float64 `json:"amount,string"`
		Price    float64 `json:"price,string"`
		Exchange string  `json:"exchange"`
		Side     string  `json:"side"`
		Type     string  `json:"type"`
		Hidden   bool    `json:"is_hidden,bool"`
	}{
		"/v1/order/new",
		strconv.FormatInt(time.Now().UnixNano(), 10),
		symbol,
		amount,
		price,
		exchange,
		side,
		otype,
		hidden,
	}

	body, err := api.post(request.URL, request)
	if err != nil {
		return order, err
	}

	err = json.Unmarshal(body, &order)
	if err != nil || order.ID == 0 { // Failed to unmarshal expected message
		// Attempt to unmarshal the error message
		errorMessage := ErrorMessage{}
		err = json.Unmarshal(body, &errorMessage)
		if err != nil { // Not expected message and not expected error, bailing...
			return order, err
		}

		return order, errors.New("API: " + errorMessage.Message)
	}

	return order, nil
}

// API GET
func (api *API) get(url string) ([]byte, error) {
	resp, err := http.Get(APIURL + url)
	if err != nil {
		return []byte{}, err
	}
	defer resp.Body.Close()

	return ioutil.ReadAll(resp.Body)
}

// API POST method based on github.com/eAndrius/bitfinex-go
func (api *API) post(url string, payload interface{}) ([]byte, error) {
	// X-BFX-PAYLOAD
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return []byte{}, err
	}
	payloadBase64 := base64.StdEncoding.EncodeToString(payloadJSON)

	// X-BFX-SIGNATURE
	h := hmac.New(sha512.New384, []byte(api.APISecret))
	h.Write([]byte(payloadBase64))
	signature := hex.EncodeToString(h.Sum(nil))

	// POST
	client := &http.Client{}
	req, err := http.NewRequest("POST", APIURL+url, nil)
	if err != nil {
		return []byte{}, err
	}

	req.Header.Add("X-BFX-APIKEY", api.APIKey)
	req.Header.Add("X-BFX-PAYLOAD", payloadBase64)
	req.Header.Add("X-BFX-SIGNATURE", signature)

	resp, err := client.Do(req)
	if err != nil {
		return []byte{}, err
	}
	defer resp.Body.Close()

	return ioutil.ReadAll(resp.Body)
}
