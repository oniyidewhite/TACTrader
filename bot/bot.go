package bot

import (
	"context"
	"errors"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/oblessing/artisgo/expert"
	"github.com/oblessing/artisgo/logger"
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
	LowPoint     float64
	HighPoint    float64
	ReadyToBuy   bool
	ReadyToShort bool
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
	// Represent the percentage change
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
			Name:   "VMA13",
			Action: VMA,
		},
	}
}

var tradeInfo = sync.Map{} // map[expert.Pair]TradeInfo{}

// ScalpingTrendTransformForTrade marks top and bottom of candle then use that information start,// Review this logic, something is still broken
func ScalpingTrendTransformForTrade(ctx context.Context, trigger expert.Candle, candles []*expert.Candle) *expert.TradeParams {
	// find first candle
	candle, err := findFirstNonNil(candles)
	if err != nil {
		logger.Error(ctx, "findFirstNonNil failed", zap.Error(err))
		return nil
	}
	// TODO: Check overall trend, do not go against this.
	// Check current market trend
	trend, data, err := evaluateMarketTrend(candles)
	if err != nil {
		logger.Error(ctx, "unable to evaluate market trend", zap.Error(err))
		return nil
	}

	// check if we can buy
	rr := getTradeInfo(candle.Pair)
	if !isNotTradeable(rr) {
		//logger.Info(ctx, "is trade-able",
		//	zap.Any("xx", rr),
		//	zap.Int("trend", trend),
		//	zap.Any("##", candle),
		//	zap.Any("trigger", trigger))
		// check if ready to long (use spread)
		// Check if it passes the existing peak in 2 candles and we are good to buy.
		// TODO: We should check if the previous candle is not greater. false flag if it is.
		if rr.ReadyToBuy &&
			trend == Green &&
			withinHSpread(ctx, rr, candle) &&
			data[1].Close <= rr.HighPoint {
			// Reset data
			rr.ReadyToBuy = false
			rr.ReadyToShort = false
			rr.LowPoint = MIN
			rr.HighPoint = MIN
			write(candle.Pair, rr)

			// long
			return &expert.TradeParams{
				TradeType:   expert.TradeTypeLong,
				OpenTradeAt: trigger.Close,
				Pair:        candle.Pair,
			}
		}
		// check if ready to short (use spread)
		if rr.ReadyToShort &&
			trend == Red &&
			withinLSpread(ctx, rr, candle) &&
			data[1].Close >= rr.LowPoint {
			// Reset data
			rr.ReadyToBuy = false
			rr.ReadyToShort = false
			rr.LowPoint = MIN
			rr.HighPoint = MIN
			write(candle.Pair, rr)

			// long
			return &expert.TradeParams{
				TradeType:   expert.TradeTypeShort,
				OpenTradeAt: trigger.Close,
				Pair:        candle.Pair,
			}
		}
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
		rr.ReadyToShort = true
		write(candle.Pair, rr)
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
		rr.ReadyToShort = false
		write(candle.Pair, rr)
	default:
		return nil
	}

	return nil
}

func withinHSpread(ctx context.Context, r TradeInfo, c expert.Candle) bool {
	return c.Close >= r.HighPoint
}

func withinLSpread(ctx context.Context, r TradeInfo, c expert.Candle) bool {
	return c.Close <= r.LowPoint
}

func isNotTradeable(info TradeInfo) bool {
	return info.HighPoint == MIN || info.LowPoint == MIN
}

func getTradeInfo(pair expert.Pair) TradeInfo {
	rr, ok := read(pair)
	if !ok {
		rr = TradeInfo{
			LowPoint:  MIN,
			HighPoint: MIN,
		}
	}

	return rr
}

func read(key expert.Pair) (TradeInfo, bool) {
	result, ok := tradeInfo.Load(key)
	if !ok {
		return TradeInfo{}, false
	}

	return result.(TradeInfo), ok
}

func write(key expert.Pair, data TradeInfo) {
	tradeInfo.Store(key, data)
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
	if len(candles) < 2 {
		return 0, errors.New("no data to evaluate")
	}

	var lowest = candles[0]
	for i := 0; i < 2; i++ {
		c := candles[i]
		if c.Close < lowest.Close {
			lowest = c
		}
	}

	return lowest.Close, nil
}

// Assumes we have only greens
func findHighest(candles []expert.Candle) (float64, error) {
	if len(candles) < 2 {
		return 0, errors.New("no data to evaluate")
	}

	var highest = candles[0]

	for i := 0; i < 2; i++ {
		c := candles[i]
		if c.Close > highest.Close {
			highest = c
		}
	}
	§
	return highest.Close, nil
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
