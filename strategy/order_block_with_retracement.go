package strategy

import (
	"context"
	"fmt"
	"github.com/oblessing/artisgo/expert"
	"sync"
)

type orderBlockWithRetracement struct {
	tradeInfo sync.Map
	size      int
}

func NewOrderBlockWithRetracement(size int) *orderBlockWithRetracement {
	return &orderBlockWithRetracement{
		tradeInfo: Store,
		size:      size,
	}
}

// TransformAndPredict finds the most recent order block
func (s *orderBlockWithRetracement) TransformAndPredict(ctx context.Context, trigger expert.Candle, candles []*expert.Candle) *expert.TradeParams {
	defer func() {
		_ = s.findAndUpdateOrderBlock(trigger.Pair, candles)
	}()

	var result *expert.TradeParams

	// check if we can buy
	res := s.getTradeInfo(trigger.Pair)
	if res.IsTradeAble() {
		if res.ReadyToBuy {
			// check if current price now touches
			if trigger.Close <= res.LowPoint {
				// long
				result = &expert.TradeParams{
					TradeType:   expert.TradeTypeLong,
					OpenTradeAt: fmt.Sprintf("%v", trigger.Close),
					Pair:        trigger.Pair,
				}

				res.ReadyToBuy = false
				res.LowPoint = MIN
				res.ReadyToShort = false
				res.HighPoint = MIN
			}
		} else if res.ReadyToShort {
			// check if current price now touches
			if trigger.Close >= res.HighPoint {
				result = &expert.TradeParams{
					TradeType:   expert.TradeTypeShort,
					OpenTradeAt: fmt.Sprintf("%v", trigger.Close),
					Pair:        trigger.Pair,
				}

				res.ReadyToShort = false
				res.HighPoint = MIN
				res.ReadyToBuy = false
				res.LowPoint = MIN
			}
		}

		s.write(trigger.Pair, res)
	}

	return result
}

func (s *orderBlockWithRetracement) read(key expert.Pair) (RSTradeInfo, bool) {
	result, ok := s.tradeInfo.Load(key)
	if !ok {
		return RSTradeInfo{
			LowPoint:     MIN,
			HighPoint:    MIN,
			ReadyToBuy:   false,
			ReadyToShort: false,
		}, false
	}

	return result.(RSTradeInfo), ok
}

func (s *orderBlockWithRetracement) write(key expert.Pair, data RSTradeInfo) {
	s.tradeInfo.Store(key, data)
}

func (s *orderBlockWithRetracement) findAndUpdateOrderBlock(key expert.Pair, candles []*expert.Candle) error {
	if len(candles) < s.size {
		return fmt.Errorf("failed precondition: min requiremt min:%v", s.size)
	}

	// For more refs: https://www.babypips.com/forexpedia/order-block
	// https://capital.com/what-is-an-order-block-in-forex
	orderBlock := candles[0]
	isGreen := isGreen(*orderBlock)
	if isGreen {
		if !allRedCandles(candles[1:]) {
			return fmt.Errorf("failed precondition: should be follwed by all reds")
		}
	} else {
		if !allGreenCandles(candles[1:]) {
			return fmt.Errorf("failed precondition: should be follwed by all greens")
		}
	}

	result, _ := s.read(key)

	if isGreen {
		result.HighPoint = orderBlock.High
		result.ReadyToShort = true
	} else {
		result.LowPoint = orderBlock.Low
		result.ReadyToBuy = true
	}

	s.write(key, result)

	return nil
}

func (s *orderBlockWithRetracement) getTradeInfo(pair expert.Pair) RSTradeInfo {
	rr, ok := s.read(pair)
	if !ok {
		rr = RSTradeInfo{
			LowPoint:  MIN,
			HighPoint: MIN,
		}
	}

	return rr
}

func allRedCandles(candles []*expert.Candle) bool {
	for _, v := range candles {
		if isGreen(*v) {
			return false
		}
	}

	return true
}

func allGreenCandles(candles []*expert.Candle) bool {
	for _, v := range candles {
		if isRed(*v) {
			return false
		}
	}

	return true
}
