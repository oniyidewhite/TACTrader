package strategy

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
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
	AdditionalData []string // minPrice, stepSize, precision
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

var MA200 = func(candles []*expert.Candle) float64 {
	size := 200
	if len(candles) < size {
		return 0
	}

	var sum float64 = 0
	for _, i := range candles {
		sum += i.Close
	}

	return sum / float64(size)
}

var MA50 = func(candles []*expert.Candle) float64 {
	size := 50
	if len(candles) < size {
		return 0
	}

	var sum float64 = 0
	for i := 1; i <= size; i++ {
		sum += candles[len(candles)-i].Close
	}

	return sum / float64(size)
}

var MA = func(candles []*expert.Candle) float64 {
	var sum float64 = 0
	for _, v := range candles {
		sum += v.Close
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

var ATR = func(candles []*expert.Candle) float64 {
	var sum float64 = 0
	for _, i := range candles {
		c, ok := i.OtherData["TR"]
		if ok {
			sum += c
		} else {
			return 0
		}
	}

	return sum / float64(len(candles))
}

var TR = func(candles []*expert.Candle) float64 {
	if len(candles) < 2 {
		return 0
	}
	prv := candles[len(candles)-2]
	lst := candles[len(candles)-1]

	return math.Max(lst.High-lst.Low, math.Max(math.Abs(lst.High-prv.Close), math.Abs(lst.Low-prv.Close)))
}

var LASTCLOSE = func(candles []*expert.Candle) float64 {
	return candles[len(candles)-1].Close
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
			Name:   "TR",
			Action: TR,
		},
		{
			Name:   "VMA",
			Action: VMA,
		},
		{
			Name:   "ATR",
			Action: ATR,
		},
	}
}

// RoundToDecimalPoint take an amount then rounds it to the upper 2 decimal point if the value is more than 2 decimal point.
func RoundToDecimalPoint(amount float64, precision uint8) float64 {
	amountString := fmt.Sprintf("%v", amount)

	amountSplit := strings.Split(amountString, ".")

	if len(amountSplit) != 2 {
		return amount
	}

	if len(amountSplit[1]) <= int(precision) {
		return amount
	}

	valueAmount := fmt.Sprintf("%s%s", amountSplit[0], amountSplit[1][:int(precision)])
	result, _ := strconv.ParseInt(valueAmount, 10, 64)

	return float64(result+1) / math.Pow(float64(10), float64(precision))
}
