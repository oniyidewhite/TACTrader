package strategy

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"go.uber.org/zap"

	"github.com/oblessing/artisgo/expert"
	"github.com/oblessing/artisgo/logger"
	"time"
)

type RSTradeInfo struct {
	LowPoint              float64
	HighPoint             float64
	ReadyToBuy            bool
	ReadyToBuyTimestamp   time.Time
	ReadyToShort          bool
	ReadyToShortTimestamp time.Time
	Metadata              string
}

func (i RSTradeInfo) IsTradeAble() bool {
	return (i.ReadyToShort && i.HighPoint != MIN) || (i.ReadyToBuy && i.LowPoint != MIN)
}

type reversalScrapingStrategy struct {
	tradeInfo sync.Map
}

func NewReversalScrapingStrategy() *reversalScrapingStrategy {
	return &reversalScrapingStrategy{
		tradeInfo: Store,
	}
}

// TransformAndPredict marks top and bottom of candle then use that information start,// Review this logic, something is still broken
func (s *reversalScrapingStrategy) TransformAndPredict(ctx context.Context, trigger expert.Candle, candles []*expert.Candle) *expert.TradeParams {
	// find first candle
	candle, err := findFirstNonNil(candles)
	if err != nil {
		logger.Error(ctx, "findFirstNonNil failed", zap.Error(err))
		return nil
	}
	// TODO: Check overall trend, do not go against this.
	// Check current market trend
	trend, data, err := s.evaluateMarketTrend(candles)
	if err != nil {
		logger.Error(ctx, "unable to evaluate market trend", zap.Error(err))
		return nil
	}

	// check if we can buy
	rr := s.getTradeInfo(candle.Pair)
	if !s.isNotTradeable(rr) {
		// logger.Info(ctx, "is trade-able",
		//	zap.Any("xx", rr),
		//	zap.Int("trend", trend),
		//	zap.Any("##", candle),
		//	zap.Any("trigger", trigger))
		// check if ready to long (use spread)
		// Check if it passes the existing peak in 2 candles and we are good to buy.
		// TODO: We should check if the previous candle is not greater. false flag if it is.
		if rr.ReadyToBuy &&
			trend == Green &&
			s.withinHSpread(ctx, rr, candle) &&
			data[1].Close <= rr.HighPoint {
			// Reset data
			rr.ReadyToBuy = false
			rr.ReadyToShort = false
			rr.LowPoint = MIN
			rr.HighPoint = MIN
			s.write(candle.Pair, rr)

			// long
			return &expert.TradeParams{
				TradeType:   expert.TradeTypeLong,
				OpenTradeAt: fmt.Sprintf("%v", trigger.Close),
				Pair:        candle.Pair,
			}
		}
		// check if ready to short (use spread)
		if rr.ReadyToShort &&
			trend == Red &&
			s.withinLSpread(ctx, rr, candle) &&
			data[1].Close >= rr.LowPoint {
			// Reset data
			rr.ReadyToBuy = false
			rr.ReadyToShort = false
			rr.LowPoint = MIN
			rr.HighPoint = MIN
			s.write(candle.Pair, rr)

			// long
			return &expert.TradeParams{
				TradeType:   expert.TradeTypeShort,
				OpenTradeAt: fmt.Sprintf("%v", trigger.Close),
				Pair:        candle.Pair,
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

		// get or create new trade info
		rr := s.getTradeInfo(candle.Pair)
		// Update highpoint
		rr.HighPoint = res
		// Since this a new high update ready to buy
		rr.ReadyToBuy = false
		rr.ReadyToShort = true
		s.write(candle.Pair, rr)
	case Red:
		res, err := s.findLowest(data)
		if err != nil {
			logger.Error(ctx, "unable to evaluate findLowest", zap.Error(err))
			return nil
		}

		// get or create new trade info
		rr := s.getTradeInfo(candle.Pair)
		// Update lowpoint
		rr.LowPoint = res
		// Since this a new high update ready to buy
		rr.ReadyToBuy = true
		rr.ReadyToShort = false
		s.write(candle.Pair, rr)
	default:
		return nil
	}

	return nil
}

func (s *reversalScrapingStrategy) withinHSpread(ctx context.Context, r RSTradeInfo, c expert.Candle) bool {
	return c.Close >= r.HighPoint
}

func (s *reversalScrapingStrategy) withinLSpread(ctx context.Context, r RSTradeInfo, c expert.Candle) bool {
	return c.Close <= r.LowPoint
}

func (s *reversalScrapingStrategy) isNotTradeable(info RSTradeInfo) bool {
	return info.HighPoint == MIN || info.LowPoint == MIN
}

func (s *reversalScrapingStrategy) getTradeInfo(pair expert.Pair) RSTradeInfo {
	rr, ok := s.read(pair)
	if !ok {
		rr = RSTradeInfo{
			LowPoint:  MIN,
			HighPoint: MIN,
		}
	}

	return rr
}

func (s *reversalScrapingStrategy) read(key expert.Pair) (RSTradeInfo, bool) {
	result, ok := s.tradeInfo.Load(key)
	if !ok {
		return RSTradeInfo{}, false
	}

	return result.(RSTradeInfo), ok
}

func (s *reversalScrapingStrategy) write(key expert.Pair, data RSTradeInfo) {
	s.tradeInfo.Store(key, data)
}

func (s *reversalScrapingStrategy) evaluateMarketTrend(candles []*expert.Candle) (Trend, []expert.Candle, error) {
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

// Assumes we have only reds
func (s *reversalScrapingStrategy) findLowest(candles []expert.Candle) (float64, error) {
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
func (s *reversalScrapingStrategy) findHighest(candles []expert.Candle) (float64, error) {
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

	return highest.Close, nil
}
