package strategy

import (
	"context"
	"fmt"
	"github.com/oblessing/artisgo/expert"
	"github.com/oblessing/artisgo/logger"
	"strings"
	"sync"
	"time"
)

type orderBlockWithTimer struct {
	tradeInfo sync.Map
	size      int
	window    []int
}

// uses latest order block then places a trade after a specific time window, works well with 3m
func NewOrderBlockWithTimer(size int, window []int) *orderBlockWithTimer {
	start, end := 13, 14
	if len(window) != 2 {
		logger.Error(context.Background(), "invalid time window provided, will default to (13 - 14)")
		return nil
	} else {
		start, end = window[0], window[1]
	}

	return &orderBlockWithTimer{
		tradeInfo: Store,
		size:      size,
		window:    []int{start, end},
	}
}

// TransformAndPredict finds the most recent order block
func (s *orderBlockWithTimer) TransformAndPredict(ctx context.Context, trigger expert.Candle, candles []*expert.Candle) *expert.TradeParams {
	defer func() {
		_ = s.findAndUpdateOrderBlock(trigger.Pair, candles)
	}()

	var result *expert.TradeParams

	// check if we can buy
	res := s.getTradeInfo(trigger.Pair)
	// 3m other block should hv a sum of not less than 1% increase (13 - 14)

	// preset action once we are in the timeframe
	if time.Now().Hour() >= s.window[0] && time.Now().Hour() < s.window[1] {
		// we just entered the time frame
		if len(res.Metadata) == 0 {
			data := ""
			if res.ReadyToBuy && res.ReadyToBuyTimestamp.After(res.ReadyToShortTimestamp) {
				data += fmt.Sprintf("%s|%v", "BUY", res.LowPoint)
			} else if res.ReadyToShort && res.ReadyToShortTimestamp.After(res.ReadyToBuyTimestamp) {
				data += fmt.Sprintf("%s|%v", "SELL", res.HighPoint)
			} else {
				return nil
			}

			res.Metadata = data
			s.write(trigger.Pair, res)
		}
	} else if len(res.Metadata) != 0 && time.Now().Hour() >= s.window[1] {
		// we are about to leave the time frame
		meta := strings.Split(res.Metadata, "|")
		if len(meta) != 2 {
			return nil
		}

		openAt := meta[1]

		if meta[0] == "BUY" {
			// long
			result = &expert.TradeParams{
				TradeType:   expert.TradeTypeLong,
				OpenTradeAt: fmt.Sprintf("%v", openAt),
				Pair:        trigger.Pair,
			}
		} else if meta[0] == "SELL" {
			result = &expert.TradeParams{
				TradeType:   expert.TradeTypeShort,
				OpenTradeAt: fmt.Sprintf("%v", openAt),
				Pair:        trigger.Pair,
			}
		}

		res.Metadata = ""
		s.write(trigger.Pair, res)
	}

	return result
}

func (s *orderBlockWithTimer) read(key expert.Pair) (RSTradeInfo, bool) {
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

func (s *orderBlockWithTimer) write(key expert.Pair, data RSTradeInfo) {
	s.tradeInfo.Store(key, data)
}

func (s *orderBlockWithTimer) findAndUpdateOrderBlock(key expert.Pair, candles []*expert.Candle) error {
	if len(candles) < s.size {
		return fmt.Errorf("failed precondition: min requiremt min:%v", s.size)
	}

	// For more refs: https://www.babypips.com/forexpedia/order-block
	// https://capital.com/what-is-an-order-block-in-forex
	// (A-B-C) (candles)
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
		// bearish
		result.HighPoint = orderBlock.High
		result.ReadyToShort = true
		result.ReadyToShortTimestamp = time.Now()
	} else {
		// bullish
		result.LowPoint = orderBlock.Low
		result.ReadyToBuy = true
		result.ReadyToBuyTimestamp = time.Now()
	}

	s.write(key, result)

	return nil
}

func (s *orderBlockWithTimer) getTradeInfo(pair expert.Pair) RSTradeInfo {
	rr, ok := s.read(pair)
	if !ok {
		rr = RSTradeInfo{
			LowPoint:  MIN,
			HighPoint: MIN,
		}
	}

	return rr
}
