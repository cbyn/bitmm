package bitfinex

import (
	"os"
	"strconv"
	"testing"
)

var APIKey = os.Getenv("BITFINEX_KEY")
var APISecret = os.Getenv("BITFINEX_SECRET")

var apiPublic = New("", "")
var apiPrivate = New(APIKey, APISecret)

func TestNewOrder(t *testing.T) {
	order, err := apiPrivate.NewOrder("btcusd", 0.1, 300, "bitfinex", "sell", "limit", true)
	if err != nil || order.ID == 0 {
		t.Error("Failed: " + err.Error())
		return
	}

	t.Log("Placed a new sell order of 0.1 btcusd @ 300 limit with ID: " + strconv.Itoa(order.ID) + ", please inspect")
}
