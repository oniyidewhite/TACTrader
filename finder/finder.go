package finder

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	settings "github.com/oblessing/artisgo"
	"github.com/oblessing/artisgo/strategy"
)

const (
	binanceAPI = "https://fapi.binance.com/fapi/v1/exchangeInfo"
)

type finderAdapter struct {
	config settings.Config
}

type Service interface {
	GetSupportedAssets(ctx context.Context) ([]strategy.PairConfig, error)
}

type CryptoPair struct {
	Symbol string `json:"symbol"`
	// IsMarginTradingAllowed bool   `json:"isMarginTradingAllowed"`
	QuotePrecision int `json:"quotePrecision"`
	Filters        []struct {
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
	var pairs []CryptoPair

	// Get from API
	if a.config.IsBypass {
		allCryptos, err2 := getPairFromFile(ctx)
		if err2 != nil {
			return []strategy.PairConfig{}, err2
		}
		pairs = allCryptos.Symbols
	} else {
		allCryptos, err2 := getPairsFromBinance(ctx)
		if err2 != nil {
			return []strategy.PairConfig{}, err2
		}
		pairs = allCryptos.Symbols
	}

	// pick only usdt pairs
	return a.filterAndMap(pairs), nil
}

func getPairFromFile(ctx context.Context) (struct {
	Symbols []CryptoPair `json:"symbols"`
}, error) {
	body, err := io.ReadAll(strings.NewReader(data))
	if err != nil {
		return struct {
			Symbols []CryptoPair `json:"symbols"`
		}{}, err
	}

	var allCryptos struct {
		Symbols []CryptoPair `json:"symbols"`
	}

	err = json.NewDecoder(strings.NewReader(string(body))).Decode(&allCryptos)
	if err != nil {
		return struct {
			Symbols []CryptoPair `json:"symbols"`
		}{}, err
	}

	return allCryptos, nil
}

func getPairsFromBinance(ctx context.Context) (struct {
	Symbols []CryptoPair `json:"symbols"`
}, error) {
	request, err := http.NewRequestWithContext(ctx, "GET", binanceAPI, nil)
	if err != nil {
		return struct {
			Symbols []CryptoPair `json:"symbols"`
		}{}, err
	}

	request.Header.Set("Connection", "keep-alive")
	request.Header.Set("User-Agent", "PostmanRuntime/7.29.2")
	request.Header.Set("Accept", "*/*")
	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		return struct {
			Symbols []CryptoPair `json:"symbols"`
		}{}, err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return struct {
			Symbols []CryptoPair `json:"symbols"`
		}{}, err
	}

	var allCryptos struct {
		Symbols []CryptoPair `json:"symbols"`
	}

	bodyString := string(body)

	err = json.NewDecoder(strings.NewReader(bodyString)).Decode(&allCryptos)
	if err != nil {
		return struct {
			Symbols []CryptoPair `json:"symbols"`
		}{}, err
	}

	return allCryptos, nil
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

	if strings.Contains(strings.ReplaceAll(input, "USDT", ""), "USD") {
		return false
	}

	// We want to trade only btc
	if !strings.EqualFold(strings.ReplaceAll(input, "USDT", ""), "BTC") {
		return false
	}

	return check == "USDT"
}

func (a finderAdapter) filterAndMap(list []CryptoPair) []strategy.PairConfig {
	var result = []strategy.PairConfig{}

	// TODO: find a better way to pass in the strategy
	algo := strategy.NewOrderBlockWithRetracement(a.config.BlockSize) // .NewJustRandom("buy") // .NewOrderBlockWithRetracement(a.config.BlockSize) // NewDivergentReversalWithRenko() // .NewWolfieStrategy(true) //

	for _, pair := range list {
		if a.isUSDT(pair.Symbol) {
			minPrice := findValueForKey("PRICE_FILTER", pair)
			stepSize := findValueForKey("LOT_SIZE", pair)
			precision := pair.QuotePrecision

			result = append(result, strategy.PairConfig{
				AdditionalData: []string{minPrice,
					stepSize, fmt.Sprintf("%v", precision)},
				Pair:            pair.Symbol,
				Period:          a.config.Interval,
				Strategy:        algo.TransformAndPredict,
				LotSize:         a.lotSize(),
				RatioToOne:      a.config.RatioToOne,
				CandleSize:      a.config.BlockSize,
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
