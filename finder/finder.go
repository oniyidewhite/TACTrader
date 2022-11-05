package finder

import (
	"context"
	"encoding/json"
	"net/http"

	settings "github.com/oblessing/artisgo"
	"github.com/oblessing/artisgo/strategy"
)

const (
	binanceAPI = "https://api.binance.com/api/v3/exchangeInfo"
)

type finderAdapter struct {
	config settings.Config
}

type Service interface {
	GetSupportedAssets(ctx context.Context) ([]strategy.PairConfig, error)
}

type CryptoPair struct {
	Symbol                 string `json:"symbol"`
	IsMarginTradingAllowed bool   `json:"isMarginTradingAllowed"`
}

func NewFinderAdapter(config settings.Config) Service {
	return finderAdapter{
		config: config,
	}
}

// GetSupportedAssets gets all the usdt pairs from binance.
func (a finderAdapter) GetSupportedAssets(ctx context.Context) ([]strategy.PairConfig, error) {
	// make api call
	request, err := http.NewRequestWithContext(ctx, "GET", binanceAPI, nil)
	if err != nil {
		return []strategy.PairConfig{}, err
	}

	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		return []strategy.PairConfig{}, err
	}

	defer resp.Body.Close()

	var allCryptos struct {
		Symbols []CryptoPair `json:"symbols"`
	}

	err = json.NewDecoder(resp.Body).Decode(&allCryptos)
	if err != nil {
		return []strategy.PairConfig{}, err
	}

	// pick only usdt pairs
	return a.filterAndMap(allCryptos.Symbols), nil
}

func (a finderAdapter) lotSize() float64 {
	return a.config.PercentageLotSize / 100
}

func (a finderAdapter) isUSDT(input string) bool {
	length := len(input) // USDT

	var check = ""
	for i := length - 4; i < length; i++ {
		check += string(input[i])
	}

	return check == "USDT"
}

func (a finderAdapter) filterAndMap(list []CryptoPair) []strategy.PairConfig {
	var result = []strategy.PairConfig{}

	// TODO: find a better way to pass in the strategy
	algo := strategy.NewReversalScrapingStrategy()

	for _, pair := range list {
		if a.isUSDT(pair.Symbol) && pair.IsMarginTradingAllowed {
			result = append(result, strategy.PairConfig{
				Pair:            pair.Symbol,
				Period:          a.config.Interval,
				Strategy:        algo.TransformAndPredict,
				LotSize:         a.lotSize(),
				RatioToOne:      3,
				CandleSize:      15,
				DefaultAnalysis: strategy.GetDefaultAnalysis(),
			})
		}
	}

	return result
}
