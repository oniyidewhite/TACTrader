package strategy

import (
	"context"
	"fmt"
	"github.com/oblessing/artisgo/expert"
	"math/rand"
	"time"
)

type justRandom struct {
	side string
}

func NewJustRandom(side string) *justRandom {
	return &justRandom{side: side}
}

// TransformAndPredict finds the most recent order block
func (s *justRandom) TransformAndPredict(ctx context.Context, trigger expert.Candle, candles []*expert.Candle) *expert.TradeParams {
	var result *expert.TradeParams

	// check if we can buy
	if s.shouldBuy() {
		result = &expert.TradeParams{
			TradeType:   expert.TradeTypeLong,
			OpenTradeAt: fmt.Sprintf("%v", trigger.Close),
			Pair:        trigger.Pair,
		}
	} else {
		result = &expert.TradeParams{
			TradeType:   expert.TradeTypeShort,
			OpenTradeAt: fmt.Sprintf("%v", trigger.Close),
			Pair:        trigger.Pair,
		}
	}

	return result
}

func (s *justRandom) shouldBuy() bool {
	switch s.side {
	case "buy":
		return true
	case "sell":
		return false
	default:
		return (1+(rand.New(rand.NewSource(time.Now().Unix())).Intn(99)))%2 == 0
	}
}
