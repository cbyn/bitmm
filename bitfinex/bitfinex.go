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
	APIURL = "https://api.bitfinex.com"
)

// Bitfinex stores Bitfinex credentials
type Bitfinex struct {
	APIKey    string
	APISecret string
}

// ErrorMessage contains an error message from exchange
type ErrorMessage struct {
	Message string `json:"message"`
}

// Book contains orderbook data from the exchange
type Book struct {
	Bids []BookItems // Slice of bid data items
	Asks []BookItems // Slice of ask data items
}

// TODO why is timestamp a float?

// BookItems contains inner orderbook data from the exchange
type BookItems struct {
	Price     float64 `json:"price,string"`     // Order price
	Amount    float64 `json:"amount,string"`    // Order volume
	Timestamp float64 `json:"timestamp,string"` // Exchange timestamp
}

// Trade contains executed trade data from the exchange
type Trade struct {
	Timestamp int     `json:"timestamp"`     // Exchange timestamp
	TID       int     `json:"tid"`           // Trade ID
	Price     float64 `json:"price,string"`  // Trade price
	Amount    float64 `json:"amount,string"` // Trade size
	Exchange  string  `json:"exchange"`      // Exchange name "bitfinex"
	Type      string  `json:"type"`          // Type, if it can be determined
}

// Trades is slice of trades
type Trades []Trade

// TODO why is timestamp a float?

// Order contains order data to/from the exchange
type Order struct {
	ID              int     `json:"id"`                         // Order ID
	Symbol          string  `json:"symbol"`                     // The symbol name the order belongs to
	Exchange        string  `json:"exchange"`                   // Exchange name "bitfinex"
	Price           float64 `json:"price,string"`               // The price the order was issued at (can be null for market orders)
	ExecutionPrice  float64 `json:"avg_execution_price,string"` // The average price at which this order as been executed so far. 0 if the order has not been executed at all
	Side            string  `json:"side"`                       // Either "buy" or "sell"
	Type            string  `json:"type"`                       // Either "market" / "limit" / "stop" / "trailing-stop"
	Timestamp       float64 `json:"timestamp,string"`           // The time the order was submitted
	IsLive          bool    `json:"is_live,bool"`               // Could the order still be filled?
	IsCancelled     bool    `json:"is_cancelled,bool"`          // Has the order been cancelled?
	WasForced       bool    `json:"was_forced,bool"`            // For margin onlytrue if it was forced by the system
	OriginalAmount  float64 `json:"original_amount,string"`     // What was the order originally submitted for?
	ExecutedAmount  float64 `json:"executed_amount,string"`     // How much of the order has been executed so far in its history?
	RemainingAmount float64 `json:"remaining_amount,string"`    // How much is still remaining to be submitted?
	Message         string  `json:"message"`                    // Message returned by some functions
	// Used in multi order responses because the API sucks
	Pair      string  `json:"pair"`             // Order symbol
	Amount    float64 `json:"amount,string"`    // Remaing order amount
	Status    string  `json:"status"`           // Order status
	CreatedAt string  `json:"created_at"`       // Creation time
	UpdatedAt string  `json:"updated_at"`       // Update time
	AvgPrice  float64 `json:"avg_price,string"` // Average execution price
}

// type multiOrder struct {
// 	ID             int     `json:"id"`                    // Order ID
// 	Pair           string  `json:"pair"`                  // Order symbol
// 	Amount         float64 `json:"amount,string"`         // Remaing order amount
// 	AvgPrice       float64 `json:"avg_price,string"`      // Average execution price
// 	OriginalAmount float64 `json:"originalamount,string"` // Original order amount
// 	CreatedAt      string  `json:"created_at"`            // Creation time
// 	Price          float64 `json:"price,string"`          // Order price
// 	Status         string  `json:"status"`                // Order status
// 	Type           string  `json:"type"`                  // Either "market" / "limit" / "stop" / "trailing-stop"
// 	UpdatedAt      string  `json:"updated_at"`            // Update time
// }

// Orders is used in processing multiple orders
type Orders struct {
	Orders  []Order `json:"order_ids"`
	Message string  `json:"message"`
}

// OrderParams contains inputs for submitting an order
type OrderParams struct {
	Symbol   string  `json:"symbol"`
	Amount   float64 `json:"amount,string"`
	Price    float64 `json:"price,string"`
	Exchange string  `json:"exchange"`
	Side     string  `json:"side"`
	Type     string  `json:"type"`
}

// Cancellation contains a response from CancelAll
type Cancellation struct {
	Result string `json:"result"`
}

// Position contains position data from the exchange
type Position struct {
	ID        int     `json:"id"`               // Position ID
	Symbol    string  `json:"symbol"`           // The symbol for the contract
	Status    string  `json:"status"`           // Status of position
	Base      float64 `json:"base,string"`      // The initiation price
	Amount    float64 `json:"amount,string"`    // Position size
	Timestamp float64 `json:"timestamp,string"` // The time the position was initiated?
	Swap      float64 `json:"swap,string"`      // ?
	PL        float64 `json:"pl,string"`        // Current PL
}

// Positions is a slice of Position
type Positions []Position

// New returns a new Bitfinex API instance
func New(key, secret string) Bitfinex {
	return Bitfinex{key, secret}
}

// Trades gets trade data from the exchange
func (bitfinex Bitfinex) Trades(symbol string, limitTrades int) (Trades, error) {
	var trades Trades

	url := fmt.Sprintf("/v1/trades/%s?limit_trades=%d", symbol, limitTrades)
	data, err := bitfinex.get(url)
	if err != nil {
		return trades, err
	}

	err = json.Unmarshal(data, &trades)
	if err != nil {
		return trades, err
	}

	return trades, nil
}

// Orderbook gets orderbook data from the exchange
func (bitfinex Bitfinex) Orderbook(symbol string, limitBids, limitAsks int) (Book, error) {
	var book Book

	url := fmt.Sprintf("/v1/book/%s?limit_bids=%d&limit_asks=%d", symbol, limitBids, limitAsks)
	data, err := bitfinex.get(url)
	if err != nil {
		return book, err
	}

	err = json.Unmarshal(data, &book)
	if err != nil {
		return book, err
	}

	return book, nil
}

// NewOrder posts new order to the exchange
func (bitfinex Bitfinex) NewOrder(symbol string, amount, price float64, exchange, side, otype string) (Order, error) {
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

	return bitfinex.postOrder(request.URL, request)
}

// MultipleNewOrders posts multiple new orders to the exchange
func (bitfinex Bitfinex) MultipleNewOrders(params []OrderParams) (Orders, error) {
	request := struct {
		URL    string        `json:"request"`
		Nonce  string        `json:"nonce"`
		Params []OrderParams `json:"orders"`
	}{
		"/v1/order/new/multi",
		strconv.FormatInt(time.Now().UnixNano(), 10),
		params,
	}

	return bitfinex.postMultiOrder(request.URL, request)
}

// CancelOrder cancels existing orders on the exchange
func (bitfinex Bitfinex) CancelOrder(id int) (Order, error) {
	request := struct {
		URL     string `json:"request"`
		Nonce   string `json:"nonce"`
		OrderID int    `json:"order_id"`
	}{
		"/v1/order/cancel",
		strconv.FormatInt(time.Now().UnixNano(), 10),
		id,
	}

	return bitfinex.postOrder(request.URL, request)
}

// CancelAll cancels all active orders
func (bitfinex Bitfinex) CancelAll() (bool, error) {
	request := struct {
		URL   string `json:"request"`
		Nonce string `json:"nonce"`
	}{
		"/v1/order/cancel/all",
		strconv.FormatInt(time.Now().UnixNano(), 10),
	}

	data, err := bitfinex.post(request.URL, request)

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

// ReplaceOrder replaces existing orders on the exchange
func (bitfinex Bitfinex) ReplaceOrder(id int, symbol string, amount, price float64, exchange, side, otype string) (Order, error) {
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

	return bitfinex.postOrder(request.URL, request)
}

// OrderStatus gets order status
func (bitfinex Bitfinex) OrderStatus(id int) (Order, error) {
	request := struct {
		URL     string `json:"request"`
		Nonce   string `json:"nonce"`
		OrderID int    `json:"order_id"`
	}{
		"/v1/order/status",
		strconv.FormatInt(time.Now().UnixNano(), 10),
		id,
	}

	return bitfinex.postOrder(request.URL, request)
}

// ActivePositions returns active positions from the exchange
func (bitfinex Bitfinex) ActivePositions() (Positions, error) {
	request := struct {
		URL   string `json:"request"`
		Nonce string `json:"nonce"`
	}{
		"/v1/positions",
		strconv.FormatInt(time.Now().UnixNano(), 10),
	}

	var positions Positions
	data, err := bitfinex.post(request.URL, request)
	if err != nil {
		return positions, err
	}

	err = json.Unmarshal(data, &positions)
	if err != nil {
		var errorMessage ErrorMessage
		err = json.Unmarshal(data, &errorMessage)
		if err != nil {
			return positions, err
		}

		return positions, errors.New(errorMessage.Message)
	}

	return positions, nil
}

// TODO: ActiveOrders

// postOrder is used in order-related API methods
func (bitfinex Bitfinex) postOrder(url string, request interface{}) (Order, error) {
	var order Order

	data, err := bitfinex.post(url, request)
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

// postMultiOrder is used in multi order-related API methods
func (bitfinex Bitfinex) postMultiOrder(url string, request interface{}) (Orders, error) {
	var orders Orders

	data, err := bitfinex.post(url, request)
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

// get executes an unauthenticated GET
func (bitfinex Bitfinex) get(url string) ([]byte, error) {
	resp, err := http.Get(APIURL + url)
	if err != nil {
		return []byte{}, err
	}
	defer resp.Body.Close()

	return ioutil.ReadAll(resp.Body)
}

// post executes an authenticated POST
func (bitfinex Bitfinex) post(url string, payload interface{}) ([]byte, error) {
	// Payload = parameters-dictionary -> JSON encode -> base64
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return []byte{}, err
	}
	payloadBase64 := base64.StdEncoding.EncodeToString(payloadJSON)

	// Signature = HMAC-SHA384(payload, api-secret) as hexadecimal
	h := hmac.New(sha512.New384, []byte(bitfinex.APISecret))
	h.Write([]byte(payloadBase64))
	signature := hex.EncodeToString(h.Sum(nil))

	client := &http.Client{}
	req, err := http.NewRequest("POST", APIURL+url, nil)
	// req.Close = true
	if err != nil {
		return []byte{}, err
	}

	// HTTP headers:
	// X-BFX-APIKEY
	// X-BFX-PAYLOAD
	// X-BFX-SIGNATURE
	req.Header.Add("X-BFX-APIKEY", bitfinex.APIKey)
	req.Header.Add("X-BFX-PAYLOAD", payloadBase64)
	req.Header.Add("X-BFX-SIGNATURE", signature)

	resp, err := client.Do(req)
	if err != nil {
		return []byte{}, err
	}
	defer resp.Body.Close()

	return ioutil.ReadAll(resp.Body)
}
