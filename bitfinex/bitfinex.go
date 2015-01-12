// Bitfinex trading API

package bitfinex

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
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
type Book struct {
	Bids []BookItems // Slice of bid data items
	Asks []BookItems // Slice of ask data items
}

// TODO: why is timestamp sometimes an int and sometimes float?
// Inner order book data from the exchange
type BookItems struct {
	Price     float64 `json:"price,string"`     // Order price
	Amount    float64 `json:"amount,string"`    // Order volume
	Timestamp float64 `json:"timestamp,string"` // Exchange timestamp
}

// TODO: why is timestamp sometimes an int and sometimes float?
// Trade data from the exchange
type Trade struct {
	Timestamp int 	  `json:"timestamp"` 		// Exchange timestamp
	TID       int     `json:"tid"`              // Trade ID
	Price     float64 `json:"price,string"`     // Trade price
	Amount    float64 `json:"amount,string"`    // Trade size
	Exchange  string  `json:"exchange"`         // Exchange name "bitfinex"
	Type      string  `json:"type"`             // Type, if it can be determined
}

// Slice of trades
type Trades []Trade

// TODO: why is timestamp sometimes an int and sometimes float?
// Order data to/from the exchange
type Order struct {
	ID              int     `json:"order_id"`                   // Order ID
	Symbol          string  `json:"symbol"`                     // The symbol name the order belongs to
	Exchange        string  `json:"exchange"`                   // Exchange name "bitfinex"
	Price           float64 `json:"price,string"`               // The price the order was issued at (can be null for market orders)
	ExecutionPrice  float64 `json:"avg_execution_price,string"` // The average price at which this order as been executed so far. 0 if the order has not been executed at all
	Side            string  `json:"side"`                       // Either "buy" or "sell"
	Type            string  `json:"type"`                       // Either "market" / "limit" / "stop" / "trailing-stop"
	Timestamp       float64 `json:"timestamp,string"`           // The timestamp the order was submitted
	IsLive          bool    `json:"is_live,bool"`               // Could the order still be filled?
	IsCancelled     bool    `json:"is_cancelled,bool"`          // Has the order been cancelled?
	WasForced       bool    `json:"was_forced,bool"`            // For margin only: true if it was forced by the system
	ExecutedAmount  float64 `json:"executed_amount,string"`     // How much of the order has been executed so far in its history?
	RemainingAmount float64 `json:"remaining_amount,string"`    // How much is still remaining to be submitted?
	OriginalAmount  float64 `json:"original_amount,string"`     // What was the order originally submitted for?
}

// Slice of orders
type Orders []Order

// Return a new Bitfinex API instance
func New(key, secret string) (api *API) {
	api = &API{
		APIKey:    key,
		APISecret: secret,
	}
	return api
}

// Get trade data from the exchange
func (api *API) Trades(symbol string, limitTrades int) (Trades, error) {
	var trades Trades

	url := fmt.Sprintf("/v1/trades/%s?limit_trades=%d", symbol, limitTrades)
	data, err := api.get(url)
	if err != nil {
		return trades, err
	}

	err = json.Unmarshal(data, &trades)
	if err != nil {
		return trades, err
	}

	return trades, nil
}

// Get orderbook data from the exchange
func (api *API) Orderbook(symbol string, limitBids, limitAsks int) (Book, error) {
	var book Book

	url := fmt.Sprintf("/v1/book/%s?limit_bids=%d&limit_asks=%d", symbol, limitBids, limitAsks)
	data, err := api.get(url)
	if err != nil {
		return book, err
	}

	err = json.Unmarshal(data, &book)
	if err != nil {
		return book, err
	}

	return book, nil
}

// Post new order to the exchange
func (api *API) NewOrder(symbol string, amount, price float64, exchange, side, otype string, hidden bool) (Order, error) {
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

	return api.postOrder(request.URL, request)
}

// Cancel existing order on the exchange
func (api *API) CancelOrder(id int) (Order, error) {
	request := struct {
		URL     string `json:"request"`
		Nonce   string `json:"nonce"`
		OrderID int    `json:"order_id"`
	}{
		"/v1/order/cancel",
		strconv.FormatInt(time.Now().UnixNano(), 10),
		id,
	}

	return api.postOrder(request.URL, request)
}

// func (api *API) ReplaceOrder() (Order, error) {
// }

// func (api *API) OrderStatus() (Order, error) {
// }

// func (api *API) ActiveOrders() (Orders, error) {
// }

// Post Order info, used in order-related API methods
func (api *API) postOrder(url string, request interface{}) (Order, error) {
	var order Order

	data, err := api.post(url, request)
	if err != nil {
		return order, err
	}

	err = json.Unmarshal(data, &order)
	if err != nil || order.ID == 0 {
		errorMessage := ErrorMessage{}
		err = json.Unmarshal(data, &errorMessage)
		if err != nil {
			return order, err
		}

		return order, errors.New("API: " + errorMessage.Message)
	}

	return order, nil
}

// API unauthenticated GET
func (api *API) get(url string) ([]byte, error) {
	resp, err := http.Get(APIURL + url)
	if err != nil {
		return []byte{}, err
	}
	defer resp.Body.Close()

	return ioutil.ReadAll(resp.Body)
}

// API autenticated POST
func (api *API) post(url string, payload interface{}) ([]byte, error) {
	// Payload = parameters-dictionary -> JSON encode -> base64
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return []byte{}, err
	}
	payloadBase64 := base64.StdEncoding.EncodeToString(payloadJSON)

	// Signature = HMAC-SHA384(payload, api-secret) as hexadecimal
	h := hmac.New(sha512.New384, []byte(api.APISecret))
	h.Write([]byte(payloadBase64))
	signature := hex.EncodeToString(h.Sum(nil))

	client := &http.Client{}
	req, err := http.NewRequest("POST", APIURL+url, nil)
	if err != nil {
		return []byte{}, err
	}

	// HTTP headers:
	// X-BFX-APIKEY
	// X-BFX-PAYLOAD
	// X-BFX-SIGNATURE
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
