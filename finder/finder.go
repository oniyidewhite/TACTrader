package finder

import (
	"context"
	"encoding/json"
	"fmt"
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
	QuotePrecision         int    `json:"quotePrecision"`
	Filters                []struct {
		FilterType string `json:"filterType"`
		MinPrice   string `json:"minPrice"`
		MaxPrice   string `json:"maxPrice"`
		TickSize   string `json:"tickSize"`
		StepSize   string `json:"stepSize"`
	} `json:"filters"`
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
	return a.config.PercentageLotSize
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
	algo := strategy.NewReversalScrapingStrategy() // NewWolfieStrategy()

	for _, pair := range list {
		if a.isUSDT(pair.Symbol) && pair.IsMarginTradingAllowed {
			minPrice := findValueForKey("PRICE_FILTER", pair)
			stepSize := findValueForKey("LOT_SIZE", pair)
			precision := pair.QuotePrecision

			result = append(result, strategy.PairConfig{
				AdditionalData:  []string{minPrice, stepSize, fmt.Sprintf("%v", precision)},
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

func findValueForKey(key string, in CryptoPair) string {
	for _, v := range in.Filters {
		if v.FilterType == key {
			result := v.TickSize
			if len(result) == 0 {
				return v.StepSize
			}
			return result
		}
	}

	return ""
}
