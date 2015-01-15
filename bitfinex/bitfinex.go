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

// Bitfinex API URL
const (
	APIURL = "https://api.bitfinex.com/"
)

// API : stores Bitfinex credentials
type API struct {
	APIKey    string
	APISecret string
}

// ErrorMessage : error message from exchange
type ErrorMessage struct {
	Message string `json:"message"`
}

// Book : orderbook data from the exchange
type Book struct {
	Bids []BookItems // Slice of bid data items
	Asks []BookItems // Slice of ask data items
}

// TODO : why is timestamp a float?

// BookItems : inner orderbook data from the exchange
type BookItems struct {
	Price     float64 `json:"price,string"`     // Order price
	Amount    float64 `json:"amount,string"`    // Order volume
	Timestamp float64 `json:"timestamp,string"` // Exchange timestamp
}

// Trade : executed trade data from the exchange
type Trade struct {
	Timestamp int     `json:"timestamp"`     // Exchange timestamp
	TID       int     `json:"tid"`           // Trade ID
	Price     float64 `json:"price,string"`  // Trade price
	Amount    float64 `json:"amount,string"` // Trade size
	Exchange  string  `json:"exchange"`      // Exchange name "bitfinex"
	Type      string  `json:"type"`          // Type, if it can be determined
}

// Trades : slice of trades
type Trades []Trade

// TODO : why is timestamp a float?

// Order : order data to/from the exchange
type Order struct {
	ID              int     `json:"id"`                         // Order ID
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

// Orders : used in processing multiple orders
type Orders struct {
	Orders []Order `json:"order_ids"`
}

// OrderParams : inputs for submitting an order
type OrderParams struct {
	Symbol   string  `json:"symbol"`
	Amount   float64 `json:"amount,string"`
	Price    float64 `json:"price,string"`
	Exchange string  `json:"exchange"`
	Side     string  `json:"side"`
	Type     string  `json:"type"`
}

// Cancellation : response from CancelAll
type Cancellation struct {
	Result string `json:"result"`
}

// New : returns a new Bitfinex API instance
func New(key, secret string) (api API) {
	api = API{
		APIKey:    key,
		APISecret: secret,
	}
	return api
}

// Trades : get trade data from the exchange
func (api API) Trades(symbol string, limitTrades int) (Trades, error) {
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

// Orderbook : get orderbook data from the exchange
func (api API) Orderbook(symbol string, limitBids, limitAsks int) (Book, error) {
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

// NewOrder : post new order to the exchange
func (api API) NewOrder(symbol string, amount, price float64, exchange, side, otype string) (Order, error) {
	request := struct {
		URL      string  `json:"request"`
		Nonce    string  `json:"nonce"`
		Symbol   string  `json:"symbol"`
		Amount   float64 `json:"amount,string"`
		Price    float64 `json:"price,string"`
		Exchange string  `json:"exchange"`
		Side     string  `json:"side"`
		Type     string  `json:"type"`
	}{
		"/v1/order/new",
		strconv.FormatInt(time.Now().UnixNano(), 10),
		symbol,
		amount,
		price,
		exchange,
		side,
		otype,
	}

	return api.postOrder(request.URL, request)
}

// MultipleNewOrders : post multiple new orders to the exchange
func (api API) MultipleNewOrders(params []OrderParams) (Orders, error) {
	request := struct {
		URL    string        `json:"request"`
		Nonce  string        `json:"nonce"`
		Params []OrderParams `json:"orders"`
	}{
		"/v1/order/new/multi",
		strconv.FormatInt(time.Now().UnixNano(), 10),
		params,
	}

	return api.postMultiOrder(request.URL, request)
}

// CancelOrder : cancel existing order on the exchange
func (api API) CancelOrder(id int) (Order, error) {
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

// CancelAll : cancel all active orders
func (api API) CancelAll() (bool, error) {
	request := struct {
		URL   string `json:"request"`
		Nonce string `json:"nonce"`
	}{
		"/v1/order/cancel/all",
		strconv.FormatInt(time.Now().UnixNano(), 10),
	}

	data, err := api.post(request.URL, request)

	var cancel Cancellation

	err = json.Unmarshal(data, &cancel)
	if err != nil {
		var errorMessage ErrorMessage
		err = json.Unmarshal(data, &errorMessage)
		if err != nil {
			return false, err
		}
		return false, errors.New(errorMessage.Message)
	}

	success := cancel.Result == "All orders cancelled"
	return success, nil
}

// ReplaceOrder : replace existing order on the exchange
func (api API) ReplaceOrder(id int, symbol string, amount, price float64, exchange, side, otype string) (Order, error) {
	request := struct {
		URL      string  `json:"request"`
		Nonce    string  `json:"nonce"`
		OrderID  int     `json:"order_id"`
		Symbol   string  `json:"symbol"`
		Amount   float64 `json:"amount,string"`
		Price    float64 `json:"price,string"`
		Exchange string  `json:"exchange"`
		Side     string  `json:"side"`
		Type     string  `json:"type"`
	}{
		"/v1/order/cancel/replace",
		strconv.FormatInt(time.Now().UnixNano(), 10),
		id,
		symbol,
		amount,
		price,
		exchange,
		side,
		otype,
	}

	return api.postOrder(request.URL, request)
}

// OrderStatus : get order status
func (api API) OrderStatus(id int) (Order, error) {
	request := struct {
		URL     string `json:"request"`
		Nonce   string `json:"nonce"`
		OrderID int    `json:"order_id"`
	}{
		"/v1/order/status",
		strconv.FormatInt(time.Now().UnixNano(), 10),
		id,
	}

	return api.postOrder(request.URL, request)
}

// postOrder : used in order-related API methods
func (api API) postOrder(url string, request interface{}) (Order, error) {
	var order Order

	data, err := api.post(url, request)
	if err != nil {
		return order, err
	}

	err = json.Unmarshal(data, &order)
	if err != nil {
		var errorMessage ErrorMessage
		err = json.Unmarshal(data, &errorMessage)
		if err != nil {
			return order, err
		}

		return order, errors.New(errorMessage.Message)
	}

	return order, nil
}

// postMultiOrder : used in multi order-related API methods
func (api API) postMultiOrder(url string, request interface{}) (Orders, error) {
	var orders Orders

	data, err := api.post(url, request)
	if err != nil {
		return orders, err
	}

	err = json.Unmarshal(data, &orders)
	if err != nil {
		var errorMessage ErrorMessage
		err = json.Unmarshal(data, &errorMessage)
		if err != nil {
			return orders, err
		}

		return orders, errors.New(errorMessage.Message)
	}

	return orders, nil
}

// get : API unauthenticated GET
func (api API) get(url string) ([]byte, error) {
	resp, err := http.Get(APIURL + url)
	if err != nil {
		return []byte{}, err
	}
	defer resp.Body.Close()

	return ioutil.ReadAll(resp.Body)
}

// post : API authenticated POST
func (api API) post(url string, payload interface{}) ([]byte, error) {
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
