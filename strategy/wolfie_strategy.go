package strategy

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/oblessing/artisgo/expert"
)

const (
	first    WState = 0
	second   WState = 1
	third    WState = 2
	fourth   WState = 3
	complete WState = 4
)

type WState int

type WTradeInfo struct {
	short WTradeData
	long  WTradeData
}

type WTradeData struct {
	state WState

	first  float64
	second float64
	third  float64
	fourth float64
}

type wolfieStrategy struct {
	tradeInfo sync.Map
}

func NewWolfieStrategy() *wolfieStrategy {
	return &wolfieStrategy{
		tradeInfo: sync.Map{},
	}
}

// TransformAndPredict builds up 4 data points of highs and lows, then predict buy and sell
func (s *wolfieStrategy) TransformAndPredict(ctx context.Context, trigger expert.Candle, candles []*expert.Candle) *expert.TradeParams {
	// Monitor long

	// Monitor short
	return nil
}

func (s *wolfieStrategy) process(candles []*expert.Candle) (expert.TradeParams, error) {
	// Check if data is valid i guess.
	c := candles[0]
	if c == nil {
		return expert.TradeParams{}, errors.New("no data")
	}

	// get trade info
	params := s.getTradeInfo(c.Pair)
	fmt.Println(params)

	// find trend
	trend := s.findTrend(candles)

	switch trend {
	case Green:

	case Red:
	}

	return expert.TradeParams{}, nil
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

func (s *wolfieStrategy) lowest(candles []*expert.Candle) (float64, error) {
	if len(candles) == 0 {
		return 0, errors.New("no data")
	}

	lowest := candles[0].Low

	for _, c := range candles {
		if lowest > c.Low {
			lowest = c.Low
		}
	}

	return lowest, nil
}

func (s *wolfieStrategy) highest(candles []*expert.Candle) (float64, error) {
	if len(candles) == 0 {
		return 0, errors.New("no data")
	}

	highest := candles[0].High

	for _, c := range candles {
		if highest < c.High {
			highest = c.High
		}
	}

	return highest, nil
}

func (s *wolfieStrategy) findTrend(candles []*expert.Candle) Trend {
	if len(candles) < 4 {
		return Unknown
	}

	var isGreen = isGreen(*candles[0]) && isGreen(*candles[1]) && isGreen(*candles[2]) && isGreen(*candles[3])
	var isRed = isRed(*candles[0]) && isRed(*candles[1]) && isRed(*candles[2]) && isRed(*candles[3])

	if isRed {
		return Red
	} else if isGreen {
		return Green
	}

	return Unknown
}
