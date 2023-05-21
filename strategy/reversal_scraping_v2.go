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

type reversalScrapingStrategyV2 struct {
	tradeInfo sync.Map
}

func NewReversalScrapingStrategyV2() *reversalScrapingStrategyV2 {
	return &reversalScrapingStrategyV2{
		tradeInfo: sync.Map{},
	}
}

// TransformAndPredict marks top and bottom of candle then use that information start,// Review this logic, something is still broken
func (s *reversalScrapingStrategyV2) TransformAndPredict(ctx context.Context, trigger expert.Candle, candles []*expert.Candle) *expert.TradeParams {
	var res *expert.TradeParams
	trend, data, err := s.evaluateMarketTrend(candles)
	if err != nil {
		logger.Error(ctx, "unable to evaluate market trend", zap.Error(err))
		return nil
	}
	// check if we can buy
	rr := s.getTradeInfo(trigger.Pair)
	if s.isTradable(rr) {
		if rr.ReadyToBuy && rr.HighPoint != MIN && s.withinHSpread(ctx, rr, trigger) {
			// Reset data
			rr.ReadyToBuy = false
			rr.HighPoint = MIN
			s.write(trigger.Pair, rr)

			// long
			res = &expert.TradeParams{
				TradeType:   expert.TradeTypeLong,
				OpenTradeAt: fmt.Sprintf("%v", trigger.Close),
				Pair:        trigger.Pair,
			}
		}
		// check if ready to short (use spread)
		if rr.ReadyToShort && rr.LowPoint != MAX && s.withinLSpread(ctx, rr, trigger) {
			// Reset data
			rr.ReadyToShort = false
			rr.LowPoint = MAX
			s.write(trigger.Pair, rr)

			// short
			res = &expert.TradeParams{
				TradeType:   expert.TradeTypeShort,
				OpenTradeAt: fmt.Sprintf("%v", trigger.Close),
				Pair:        trigger.Pair,
			}
		}
	}

	// Updated data based on trend & mark ready to buy based on trend.
	switch trend {
	case Green:
		res, err := s.findHighest(data)
		if err != nil {
			logger.Error(ctx, "unable to evaluate findHighest", zap.Error(err))
			return nil
		}

		rr.HighPoint = res
		rr.ReadyToBuy = false
		s.write(trigger.Pair, rr)
	case Red:
		res, err := s.findLowest(data)
		if err != nil {
			logger.Error(ctx, "unable to evaluate findLowest", zap.Error(err))
			return nil
		}

		// Update lowpoint
		rr.LowPoint = res
		rr.ReadyToShort = false
		s.write(trigger.Pair, rr)
	default:
		if rr.HighPoint > trigger.Close {
			rr.ReadyToBuy = true
		}

		if rr.LowPoint < trigger.Close {
			rr.ReadyToShort = true
		}

		s.write(trigger.Pair, rr)
	}

	return res
}

func (s *reversalScrapingStrategyV2) evaluateMarketTrend(candles []*expert.Candle) (Trend, []expert.Candle, error) {
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

func (s *reversalScrapingStrategyV2) withinHSpread(ctx context.Context, r RSTradeInfo, c expert.Candle) bool {
	return c.Close >= r.HighPoint
}

func (s *reversalScrapingStrategyV2) withinLSpread(ctx context.Context, r RSTradeInfo, c expert.Candle) bool {
	return c.Close <= r.LowPoint
}

func (s *reversalScrapingStrategyV2) isTradable(info RSTradeInfo) bool {
	return info.HighPoint != MIN || info.LowPoint != MAX
}

func (s *reversalScrapingStrategyV2) getTradeInfo(pair expert.Pair) RSTradeInfo {
	rr, ok := s.read(pair)
	if !ok {
		rr = RSTradeInfo{
			LowPoint:  MAX,
			HighPoint: MIN,
		}
	}

	return rr
}

func (s *reversalScrapingStrategyV2) read(key expert.Pair) (RSTradeInfo, bool) {
	result, ok := s.tradeInfo.Load(key)
	if !ok {
		return RSTradeInfo{}, false
	}

	return result.(RSTradeInfo), ok
}

func (s *reversalScrapingStrategyV2) write(key expert.Pair, data RSTradeInfo) {
	s.tradeInfo.Store(key, data)
}

// Assumes we have only reds
func (s *reversalScrapingStrategyV2) findLowest(candles []expert.Candle) (float64, error) {
	if len(candles) < 2 {
		return 0, errors.New("no data to evaluate")
	}

	var lowest = candles[0]
	for i := len(candles) - 2; i < len(candles); i++ {
		c := candles[i]
		if c.Close < lowest.Close {
			lowest = c
		}
	}

	return lowest.Close, nil
}

// Assumes we have only greens
func (s *reversalScrapingStrategyV2) findHighest(candles []expert.Candle) (float64, error) {
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
