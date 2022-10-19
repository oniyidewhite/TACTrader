package finder

import (
	"context"
	"encoding/json"
	"net/http"

	TACTrader "github.com/oblessing/artisgo"
	trade "github.com/oblessing/artisgo/bot"
)

const (
	binanceAPI  = "https://api.binance.com/api/v3/exchangeInfo"
)

type CryptoPair struct {
	Symbol string `json:"symbol"`
	IsMarginTradingAllowed bool `json:"isMarginTradingAllowed"`
}

func lotSize() float64 {
	return TACTrader.PercentageLotSize / 100
}

// GetAllUsdtPairs gets all the usdt pairs from binance.
func GetAllUsdtPairs(ctx context.Context) ([]trade.PairConfig, error) {
	// make api call
	request, err := http.NewRequestWithContext(ctx, "GET", binanceAPI, nil)
	if err != nil {
		return []trade.PairConfig{}, err
	}

	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		return []trade.PairConfig{}, err
	}

	defer resp.Body.Close()

	var allCryptos struct{
		Symbols []CryptoPair `json:"symbols"`
	}

	err = json.NewDecoder(resp.Body).Decode(&allCryptos)
	if err != nil {
		return []trade.PairConfig{}, err
	}

	// pick only usdt pairs
	return filterAndMap(allCryptos.Symbols), nil
}

func isUSDT(input string) bool {
	length := len(input) // USDT

	var check = ""
	for i := length - 4; i < length; i++ {
		check += string(input[i])
	}

	return check == "USDT"
}

func filterAndMap(list []CryptoPair) []trade.PairConfig {
	var result = []trade.PairConfig{}

	for _, pair := range list {
		if  isUSDT(pair.Symbol) && pair.IsMarginTradingAllowed  { // pair.Symbol == "BTCUSDT" || pair.Symbol == "ETHUSDT"
			result = append(result, trade.PairConfig{
				Pair:           pair.Symbol,
				Period:         TACTrader.Interval,
				Strategy:       trade.ScalpingTrendTransformForTrade,
				OverrideParams: true,
				LotSize:        lotSize(), // 2%, calculate when we get a change of candle
				RatioToOne:     3,
				TradeSize:      "",//fmt.Sprintf("%f", (1/pair.Price())*tradeAmount), // calculate when we are about to make trade
			})
		}
	}

	return result
}
