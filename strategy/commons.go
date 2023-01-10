package strategy

import (
	"context"
	"errors"
	"time"

	"github.com/oblessing/artisgo/expert"
)

const (
	MIN float64 = -1 << 63
	MAX float64 = 1 << 63
)

const (
	Unknown Trend = iota
	Red
	Green
)

type Trend int

type AlgoStrategy interface {
	TransformAndPredict(ctx context.Context, trigger expert.Candle, candles []*expert.Candle) *expert.TradeParams
}

type Candle struct {
	Pair  float64
	Open  float64
	Close float64
	High  float64
	Low   float64
	Vol   float64
	Time  time.Time
}

// PairConfig represent a crypto pair configuration
type PairConfig struct {
	QuotePrecision int
	Pair           string
	Period         string
	Strategy       expert.Transform
	// Represent the percentage change
	LotSize         float64
	RatioToOne      float64
	DisableStopLoss bool
	// DefaultAnalysis contains other this we should monitor for this symbol
	DefaultAnalysis []*expert.CalculateAction
	// CandleStick size
	CandleSize int
}

// RSI 66.6(), 33.3
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
	size := float64(len(candles) - 1)
	rsi := 100 - (100 / (1 + ((sumUp / size) / (sumDown / size))))
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

func isGreen(candle expert.Candle) bool {
	return candle.IsUp()
}

func isRed(candle expert.Candle) bool {
	return !candle.IsUp()
}

func findFirstNonNil(candles []*expert.Candle) (expert.Candle, error) {
	var data *expert.Candle

	for _, c := range candles {
		if data != nil {
			break
		}

		data = c
	}

	if data == nil {
		return expert.Candle{}, errors.New("invalid data resolved")
	}

	return *data, nil
}

func GetDefaultAnalysis() []*expert.CalculateAction {
	return []*expert.CalculateAction{
		{
			Name:   "MA",
			Action: MA,
		},
		{
			Name:   "RSI",
			Action: RSI,
		},
		{
			Name:   "VMA",
			Action: VMA,
		},
	}
}
