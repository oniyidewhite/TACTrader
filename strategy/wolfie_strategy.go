package strategy

import (
	"context"
	"fmt"
	"sync"

	"github.com/oblessing/artisgo/expert"
)

type WTradeInfo struct {
	LowPoint  float64
	HighPoint float64
}

type wolfieStrategy struct {
	tradeInfo sync.Map
	useV2     bool
}

func NewWolfieStrategy(v2 bool) *wolfieStrategy {
	return &wolfieStrategy{
		tradeInfo: sync.Map{},
		useV2:     v2,
	}
}

// TransformAndPredict builds up 4 data points of highs and lows, then predict buy and sell
func (s *wolfieStrategy) TransformAndPredict(ctx context.Context, trigger expert.Candle, candles []*expert.Candle) *expert.TradeParams {
	if !s.useV2 {
		res := DetectWolfePattern(candles)

		if res.DownTrend != nil {
			rr := s.getTradeInfo(trigger.Pair)
			rr.LowPoint = MIN
			rr.HighPoint = MIN
			s.write(trigger.Pair, rr)

			return &expert.TradeParams{
				TradeType:   expert.TradeTypeShort,
				OpenTradeAt: fmt.Sprintf("%f", trigger.Close),
				Pair:        trigger.Pair,
			}
		} else if res.UpTrend != nil {
			rr := s.getTradeInfo(trigger.Pair)
			rr.LowPoint = MIN
			rr.HighPoint = MIN
			s.write(trigger.Pair, rr)

			return &expert.TradeParams{
				TradeType:   expert.TradeTypeLong,
				OpenTradeAt: fmt.Sprintf("%f", trigger.Close),
				Pair:        trigger.Pair,
			}
		}

		return nil
	}

	// v2
	// Monitor long
	shouldTrade, direction := DetectWolfePatternAndDirection(candles)
	if !shouldTrade {
		return nil
	}

	if direction == Bullish {
		rr := s.getTradeInfo(trigger.Pair)
		rr.LowPoint = MIN
		rr.HighPoint = MIN
		s.write(trigger.Pair, rr)

		return &expert.TradeParams{
			TradeType:   expert.TradeTypeLong,
			OpenTradeAt: fmt.Sprintf("%f", trigger.Close),
			Pair:        trigger.Pair,
		}
	} else if direction == Bearish {
		rr := s.getTradeInfo(trigger.Pair)
		rr.LowPoint = MIN
		rr.HighPoint = MIN
		s.write(trigger.Pair, rr)

		return &expert.TradeParams{
			TradeType:   expert.TradeTypeShort,
			OpenTradeAt: fmt.Sprintf("%f", trigger.Close),
			Pair:        trigger.Pair,
		}
	}

	return nil
}

func (s *wolfieStrategy) getTradeInfo(pair expert.Pair) WTradeInfo {
	rr, ok := s.read(pair)
	if !ok {
		rr = WTradeInfo{}
	}

	return rr
}

func (s *wolfieStrategy) write(key expert.Pair, data WTradeInfo) {
	s.tradeInfo.Store(key, data)
}

func (s *wolfieStrategy) read(key expert.Pair) (WTradeInfo, bool) {
	result, ok := s.tradeInfo.Load(key)
	if !ok {
		return WTradeInfo{}, false
	}

	return result.(WTradeInfo), ok
}

type TrendX struct {
	Start  int
	End    int
	Type   string
	Points []*expert.Candle
}

type WolfePattern struct {
	UpTrend   *TrendX
	DownTrend *TrendX
}

func calculateTrend(points []*expert.Candle, length int, trendType string) *TrendX {
	//var trend TrendX
	trendTypePrefix := "Up"
	if trendType == "down" {
		trendTypePrefix = "Down"
	}
	var highestHigh, lowestLow float64
	var highestHighIndex, lowestLowIndex int
	for i := length; i < len(points); i++ {
		for j := i - length; j <= i; j++ {
			if points[j].High > highestHigh || highestHigh == 0 {
				highestHigh = points[j].High
				highestHighIndex = j
			}
			if points[j].Low < lowestLow || lowestLow == 0 {
				lowestLow = points[j].Low
				lowestLowIndex = j
			}
		}
		if highestHighIndex < lowestLowIndex {
			break
		}
	}
	if highestHighIndex > 0 && lowestLowIndex > 0 {
		return &TrendX{
			Start: highestHighIndex - length,
			End:   lowestLowIndex,
			Type:  trendTypePrefix + "Trend",
			//Points: points[lowestLowIndex+1 : highestHighIndex-length],
		}
	}
	return nil
}

func DetectWolfePattern(points []*expert.Candle) WolfePattern {
	var upTrend, downTrend *TrendX
	upTrend = calculateTrend(points, 5, "up")
	downTrend = calculateTrend(points, 5, "down")
	return WolfePattern{
		UpTrend:   upTrend,
		DownTrend: downTrend,
	}
}

// WolfePatternDirection represents the direction of a Wolfe pattern
type WolfePatternDirection int

const (
	Bullish WolfePatternDirection = iota
	Bearish
)

// DetectWolfePatternAndDirection detects Wolfe patterns in a slice of candlesticks
// Returns true and the direction if a Wolfe pattern is found, false and an empty direction otherwise
func DetectWolfePatternAndDirection(data []*expert.Candle) (bool, WolfePatternDirection) {
	if len(data) < 10 {
		return false, 0
	}
	var upTrend, downTrend bool
	var lastHigh, lastLow float64
	var point2, _, _ float64
	for i := 4; i < len(data)-1; i++ {
		// Check for an up trend
		if data[i].Close > data[i-2].Close && data[i-2].Close > data[i-4].Close {
			upTrend = true
			lastHigh = data[i].High
			// Check for point 2
			if lastLow > 0 && data[i-2].Low < lastLow {
				point2 = data[i-2].Low
			}
		} else if data[i].Close < data[i-2].Close && data[i-2].Close < data[i-4].Close {
			downTrend = true
			lastLow = data[i].Low
			// Check for point 2
			if lastHigh > 0 && data[i-2].High > lastHigh {
				point2 = data[i-2].High
			}
		}
		// Check for point 3 and point 4
		if upTrend && downTrend {
			if point2 > 0 && data[i-1].Low < point2 {
				_ = data[i-1].Low
				_ = data[i].High
				return true, Bullish
			} else if point2 > 0 && data[i-1].High > point2 {
				_ = data[i-1].High
				_ = data[i].Low
				return true, Bearish
			}
			upTrend = false
			downTrend = false
		}
	}
	return false, 0
}
