package bot

import (
	"context"
	"errors"
	"github.com/oblessing/artisgo/expert"
	"github.com/oblessing/artisgo/logger"
	"go.uber.org/zap"
	"time"
)

const (
	MIN float64 = -1 << 63
	MAX float64 = 1 << 63
)

const (
	Unknown = 1 + iota
	Red
	Green
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

type TradeInfo struct {
	LowPoint   float64
	HighPoint  float64
	ReadyToBuy bool
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
	Pair       string
	Period     string
	TradeSize  string // To buy or short.
	Strategy   expert.Transform
	LotSize    float64
	RatioToOne float64
	Spread     float64
	// Override expert stop & take profit with config info
	OverrideParams  bool
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

var tradeInfo = map[expert.Pair]TradeInfo{}

// ScalpingTrendTransformForBuy marks top and bottom of candle then use that information start
func ScalpingTrendTransformForBuy(ctx context.Context, candles []*expert.Candle) *expert.TradeParams {
	// find first candle
	candle, err := findFirstNonNil(candles)
	if err != nil {
		logger.Error(ctx, "findFirstNonNil failed", zap.Error(err))
		return nil
	}
	// if not candle.closed
	if !candle.Closed {
		rr := getTradeInfo(candle.Pair)
		if isNotTradeable(rr) {
			return nil
		}

		// check if ready to buy (use spread)
		if withinSpread(ctx, rr, candle) && rr.ReadyToBuy {
			// buy
			rr.ReadyToBuy = false
			tradeInfo[candle.Pair] = rr
			return &expert.TradeParams{
				OpenTradeAt: rr.HighPoint,
				Rating:      30, // Good to buy
				Pair:        candle.Pair,
			}
		}

		return nil
	}

	// Check market trend
	trend, data, err := evaluateMarketTrend(candles)
	if err != nil {
		logger.Error(ctx, "unable to evaluate market trend", zap.Error(err))
		return nil
	}

	// Updated data based on trend & mark ready to buy based on trend.
	switch trend {
	case Green:
		res, err := findHighest(data)
		if err != nil {
			logger.Error(ctx, "unable to evaluate findHighest", zap.Error(err))
			return nil
		}

		// get or create new trade info
		rr := getTradeInfo(candle.Pair)
		// Update highpoint
		rr.HighPoint = res
		// Since this a new high update ready to buy
		rr.ReadyToBuy = false
		tradeInfo[candle.Pair] = rr
	case Red:
		res, err := findLowest(data)
		if err != nil {
			logger.Error(ctx, "unable to evaluate findLowest", zap.Error(err))
			return nil
		}

		// get or create new trade info
		rr := getTradeInfo(candle.Pair)
		// Update lowpoint
		rr.LowPoint = res
		// Since this a new high update ready to buy
		rr.ReadyToBuy = true
		tradeInfo[candle.Pair] = rr
	default:
		return nil
	}

	return nil
}

func withinSpread(ctx context.Context, r TradeInfo, c expert.Candle) bool {
	bound := ctx.Value("spread").(float64)

	return c.Close >= r.HighPoint && c.Close <= (r.HighPoint+bound)
}

func isNotTradeable(info TradeInfo) bool {
	return info.HighPoint == MIN || info.LowPoint == MIN
}

func getTradeInfo(pair expert.Pair) TradeInfo {
	rr, ok := tradeInfo[pair]
	if !ok {
		rr = TradeInfo{
			LowPoint:  MIN,
			HighPoint: MIN,
		}
	}

	return rr
}

func evaluateMarketTrend(candles []*expert.Candle) (int, []expert.Candle, error) {
	var last2 []expert.Candle
	for _, c := range candles {
		if len(last2) == 2 {
			break
		}

		if c != nil {
			last2 = append(last2, *c)
		}
	}

	if len(last2) != 2 {
		return Unknown, last2, errors.New("not enough data")
	}

	var isGreen = isGreen(last2[0]) && isGreen(last2[1])
	var isRed = isRed(last2[0]) && isRed(last2[1])

	if isRed {
		return Red, last2, nil
	} else if isGreen {
		return Green, last2, nil
	}

	return Unknown, last2, nil
}

func isGreen(candle expert.Candle) bool {
	return candle.IsUp()
}

func isRed(candle expert.Candle) bool {
	return !candle.IsUp()
}

// Assumes we have only reds
func findLowest(candles []expert.Candle) (float64, error) {
	var lowest *expert.Candle

	for _, c := range candles {
		if lowest == nil || lowest.Close > c.Close {
			lowest = &c
		}
	}

	if lowest != nil {
		return lowest.Close, nil
	}

	return 0, errors.New("no data to evaluate")
}

// Assumes we have only greens
func findHighest(candles []expert.Candle) (float64, error) {
	var highest *expert.Candle

	for _, c := range candles {
		if highest == nil || highest.Close < c.Close {
			highest = &c
		}
	}

	if highest != nil {
		return highest.Close, nil
	}

	return 0, errors.New("no data to evaluate")
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
