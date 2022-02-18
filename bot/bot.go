package bot

import (
	"github.com/oblessing/artisgo/expert"
	"time"
)

const (
	MIN float64 = -1 << 63
	MAX float64 = 1 << 63
)

type Candle struct {
	Pair  float64
	Open  float64
	Close float64
	High  float64
	Low   float64
	Vol   float64
	Time  time.Time
}

type Args struct {
	Time       time.Time
	Pair       string
	Open       float64
	StopLoss   float64
	TakeProfit float64
}

// PairConfig represent a crypto pair configuration
type PairConfig struct {
	Pair            string
	Period          string
	TradeSize       string // To buy or short.
	Strategy        expert.Transform
	DisableStopLoss bool
}

type Trader interface {
	// WatchAndTrade a PairConfig you want to trade. returns error if unable or trader has started.
	WatchAndTrade(...PairConfig) error
	// StartTrading this trader. note: blocking call
	StartTrading() error
}

var RSI = func(candles []*expert.Candle) float64 {
	var sumUp float64 = 0
	var sumDown float64 = 0
	for _, i := range candles {
		if i.IsUp() {
			sumUp += i.Close - i.Open
		} else {
			sumDown += i.Open - i.Close
		}
	}
	rsi := 100 - (100 / (1 - (sumUp / sumDown)))
	return rsi
}

var MA = func(candles []*expert.Candle) float64 {
	var sum float64 = 0
	for _, i := range candles {
		sum += i.Close
	}

	return sum / float64(len(candles))
}

var VMA = func(candles []*expert.Candle) float64 {
	var sum float64 = 0
	for _, i := range candles {
		sum += i.Volume
	}

	return sum / float64(len(candles))
}

var VRSI = func(candles []*expert.Candle) float64 {
	var sumUp float64 = 0
	var sumDown float64 = 0
	for _, i := range candles {
		if i.IsUp() {
			sumUp += i.Volume
		} else {
			sumDown += i.Volume
		}
	}
	rsi := 100 - (100 / (1 - (sumUp / sumDown)))
	return rsi
}

func GetDefaultAnalysis() []*expert.CalculateAction {
	return []*expert.CalculateAction{
		{
			Name:   "MA36",
			Size:   36,
			Action: MA,
		},
		{
			Name:   "RSI6",
			Size:   6,
			Action: RSI,
		},
		{
			Name:   "RSI14",
			Size:   14,
			Action: RSI,
		},
		{
			Name:   "VMA18",
			Size:   18,
			Action: VMA,
		},
		{
			Name:   "VRSI18",
			Size:   18,
			Action: VRSI,
		},
	}
}

// TTMyCustomTransform uses an array of candles find the lowest and highest that calculate it's percentage to the current
// returns that result as it's rating
// Use the MA to determine it it's good buy or not
func TTMyCustomTransform(candles []*expert.Candle) *expert.TradeParams {
	//

	return &expert.TradeParams{}
}

// MyOldCustomTransform uses an array of candles find the lowest and highest that calculate it's percentage to the current
// returns that result as it's rating
// Use the MA to determine it it's good buy or not
func MyOldCustomTransform(candles []*expert.Candle) *expert.TradeParams {
	high1, high2 := MIN, MIN
	low1, low2 := MAX, MAX

	current := candles[0].Close

	for _, c := range candles {
		if c.Low < low1 {
			low2 = low1
			low1 = c.Low
		}

		if c.High > high1 {
			high2 = high1
			high1 = c.High
		}
	}

	_ = (high1 + high2) / 2
	_ = (low1 + low2) / 2

	rate := ((current - low1) / (high1 - low1)) * 100

	result := &expert.TradeParams{
		OpenTradeAt:  current,
		TakeProfitAt: high1,
		StopLossAt:   low1,
		Rating:       int(rate),
		Pair:         candles[0].Pair,
	}
	return result
}
