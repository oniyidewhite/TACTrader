package strategy

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"go.uber.org/zap"

	"github.com/oblessing/artisgo/expert"
	"github.com/oblessing/artisgo/logger"
)

type divergentReversalWithRenko struct {
	tradeInfo sync.Map
}

func NewDivergentReversalWithRenko() *divergentReversalWithRenko {
	return &divergentReversalWithRenko{
		tradeInfo: Store,
	}
}

// TransformAndPredict marks top and bottom of candle then use that information start,// Review this logic, something is still broken
func (s *divergentReversalWithRenko) TransformAndPredict(ctx context.Context, trigger expert.Candle, candles []*expert.Candle) *expert.TradeParams {
	var res *expert.TradeParams
	trend, _, err := s.findDivergent(candles)
	if err != nil {
		logger.Error(ctx, "unable to evaluate market trend", zap.Error(err))
		return nil
	}

	// Updated data based on trend & mark ready to buy based on trend.
	switch trend {
	case Green:
		res = &expert.TradeParams{
			TradeType:   expert.TradeTypeLong,
			OpenTradeAt: fmt.Sprintf("%v", trigger.Close),
			Pair:        trigger.Pair,
		}
	case Red:
		res = &expert.TradeParams{
			TradeType:   expert.TradeTypeShort,
			OpenTradeAt: fmt.Sprintf("%v", trigger.Close),
			Pair:        trigger.Pair,
		}
	}

	return res
}

func (s *divergentReversalWithRenko) evaluateMarketTrend(candles []*expert.Candle) (Trend, []expert.Candle, error) {
	var data []expert.Candle
	for _, c := range candles {
		if c != nil {
			data = append(data, *c)
		}
	}

	var isGreen = isGreen(data[len(data)-1]) && isGreen(data[len(data)-2])
	var isRed = isRed(data[len(data)-1]) && isRed(data[len(data)-2])

	if isRed {
		return Red, data, nil
	} else if isGreen {
		return Green, data, nil
	}

	return Unknown, data, nil
}

func (s *divergentReversalWithRenko) withinHSpread(ctx context.Context, r RSTradeInfo, c expert.Candle) bool {
	return c.Close > r.HighPoint
}

func (s *divergentReversalWithRenko) withinLSpread(ctx context.Context, r RSTradeInfo, c expert.Candle) bool {
	return c.Close < r.LowPoint
}

func (s *divergentReversalWithRenko) isTradable(info RSTradeInfo) bool {
	return info.HighPoint != MIN || info.LowPoint != MAX
}

func (s *divergentReversalWithRenko) getTradeInfo(pair expert.Pair) RSTradeInfo {
	rr, ok := s.read(pair)
	if !ok {
		rr = RSTradeInfo{
			LowPoint:  MAX,
			HighPoint: MIN,
		}
	}

	return rr
}

func (s *divergentReversalWithRenko) read(key expert.Pair) (RSTradeInfo, bool) {
	result, ok := s.tradeInfo.Load(key)
	if !ok {
		return RSTradeInfo{}, false
	}

	return result.(RSTradeInfo), ok
}

func (s *divergentReversalWithRenko) write(key expert.Pair, data RSTradeInfo) {
	s.tradeInfo.Store(key, data)
}

// Assumes we have only reds
func (s *divergentReversalWithRenko) findLowest(candles []expert.Candle) (float64, error) {
	if len(candles) < 2 {
		return 0, errors.New("no data to evaluate")
	}

	var lowest = candles[len(candles)-2]
	for i := len(candles) - 2; i < len(candles); i++ {
		c := candles[i]
		if c.Close < lowest.Close {
			lowest = c
		}
	}

	return lowest.Close, nil
}

// Assumes we have only greens
func (s *divergentReversalWithRenko) findHighest(candles []expert.Candle) (float64, error) {
	if len(candles) < 2 {
		return 0, errors.New("no data to evaluate")
	}

	var highest = candles[len(candles)-2]

	for i := len(candles) - 2; i < len(candles); i++ {
		c := candles[i]
		if c.Close > highest.Close {
			highest = c
		}
	}

	return highest.Close, nil
}

// Data starts here

func (s *divergentReversalWithRenko) findDivergent(candles []*expert.Candle) (Trend, []expert.Candle, error) {
	if len(candles) < 2 {
		return Unknown, nil, errors.New("no data to evaluate")
	}

	// check if about to go down
	hc := findHighestCandle(candles)
	hcrsi := findHighestRsiCandle(candles)
	// check if about to go down
	lc := findLowestCandle(candles)
	lcrsi := findLowestRsiCandle(candles)

	// check if same candle
	long := hcrsi.OtherData["RSI"] > hc.OtherData["RSI"] && candles[len(candles)-1].Time == hc.Time
	short := lcrsi.OtherData["RSI"] < lc.OtherData["RSI"] && candles[len(candles)-1].Time == lc.Time
	if long && short {
		return Unknown, cleanupCandle(candles), nil
	} else if long {
		return Red, cleanupCandle(candles), nil
	} else if short {
		return Green, cleanupCandle(candles), nil
	} else {
		return Unknown, cleanupCandle(candles), nil
	}
}

func findLowestCandle(candles []*expert.Candle) expert.Candle {
	var lowest = candles[0]
	for _, c := range candles {
		if c.Close < lowest.Close {
			lowest = c
		}
	}

	return *lowest
}

func findLowestRsiCandle(candles []*expert.Candle) expert.Candle {
	var highest = candles[0]
	for _, c := range candles {
		if c.OtherData["RSI"] < highest.OtherData["RSI"] {
			highest = c
		}
	}

	return *highest
}

func findHighestCandle(candles []*expert.Candle) expert.Candle {
	var highest = candles[0]
	for _, c := range candles {
		if c.Close > highest.Close {
			highest = c
		}
	}

	return *highest
}

func findHighestRsiCandle(candles []*expert.Candle) expert.Candle {
	var highest = candles[0]
	for _, c := range candles {
		if c.OtherData["RSI"] > highest.OtherData["RSI"] {
			highest = c
		}
	}

	return *highest
}

func cleanupCandle(candles []*expert.Candle) []expert.Candle {
	var data []expert.Candle

	for _, c := range candles {
		d := c

		data = append(data, *d)
	}

	return data
}
