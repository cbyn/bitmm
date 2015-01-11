// Package to communicate with bitfinex exchange

package bitfinex

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
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
func (api *API) GetBook(url string) Orderbook {
	var book Orderbook
	data, err := api.get("book/" + url)
	if err != nil {
		log.Fatal(err)
	}
	err = json.Unmarshal(data, &book)
	if err != nil {
		log.Fatal(err)
	}

	return book
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
	// parameters-dictionary -> JSON encode -> base64
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return []byte{}, err
	}
	payloadBase64 := base64.StdEncoding.EncodeToString(payloadJSON)

	// X-BFX-SIGNATURE
	// HMAC-SHA384(payload, api-secret) as hexadecimal
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
