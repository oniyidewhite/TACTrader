package strategy

import (
	"context"
	"github.com/oblessing/artisgo/expert"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestNewReversalScrapingStrategyV2(t *testing.T) {

	t.Run("test short func", func(t *testing.T) {
		adpt := NewReversalScrapingStrategyV2()

		candles := []*expert.Candle{
			{
				Pair:      "TEST",
				High:      11,
				Low:       5,
				Open:      10,
				Close:     5,
				Volume:    0,
				OtherData: map[string]float64{},
				Time:      time.Now().Unix(),
				Closed:    true,
			},
			{
				Pair:      "TEST",
				High:      11,
				Low:       2,
				Open:      5,
				Close:     3,
				Volume:    0,
				OtherData: map[string]float64{},
				Time:      time.Now().Unix(),
				Closed:    true,
			},
		}

		t.Run("should update lowest point", func(t *testing.T) {

			ctx := context.Background()

			trigger := expert.Candle{
				Pair:      "TEST",
				High:      11,
				Low:       2,
				Open:      3,
				Close:     4,
				Volume:    0,
				OtherData: map[string]float64{},
				Time:      time.Now().Unix(),
				Closed:    true,
			}

			// should update the lowest point
			res := adpt.TransformAndPredict(ctx, trigger, candles)
			assert.Nil(t, res)
			d, ok := adpt.tradeInfo.Load(expert.Pair("TEST"))
			assert.True(t, ok)
			assert.Equal(t, float64(3), d.(RSTradeInfo).LowPoint)
		})

		// requires first to completely update
		t.Run("should indicated ready to short", func(t *testing.T) {
			ctx := context.Background()

			candles = append(candles, &expert.Candle{
				Pair:      "TEST",
				High:      11,
				Low:       2,
				Open:      3,
				Close:     4,
				Volume:    0,
				OtherData: map[string]float64{},
				Time:      time.Now().Unix(),
				Closed:    true,
			})

			trigger := expert.Candle{
				Pair:      "TEST",
				High:      11,
				Low:       2,
				Open:      5,
				Close:     4,
				Volume:    0,
				OtherData: map[string]float64{},
				Time:      time.Now().Unix(),
				Closed:    true,
			}

			// should update the lowest point
			res := adpt.TransformAndPredict(ctx, trigger, candles)
			assert.Nil(t, res)
			d, ok := adpt.tradeInfo.Load(expert.Pair("TEST"))
			assert.True(t, ok)
			assert.Equal(t, float64(3), d.(RSTradeInfo).LowPoint)
			assert.True(t, d.(RSTradeInfo).ReadyToShort)
		})

		// requires first 2 to completely update
		t.Run("should short", func(t *testing.T) {
			ctx := context.Background()

			candles = append(candles, &expert.Candle{
				Pair:      "TEST",
				High:      11,
				Low:       2,
				Open:      4,
				Close:     3,
				Volume:    0,
				OtherData: map[string]float64{},
				Time:      time.Now().Unix(),
				Closed:    true,
			},
				&expert.Candle{
					Pair:      "TEST",
					High:      11,
					Low:       2,
					Open:      3,
					Close:     2,
					Volume:    0,
					OtherData: map[string]float64{},
					Time:      time.Now().Unix(),
					Closed:    true,
				},
			)

			trigger := expert.Candle{
				Pair:      "TEST",
				High:      11,
				Low:       2,
				Open:      2,
				Close:     3,
				Volume:    0,
				OtherData: map[string]float64{},
				Time:      time.Now().Unix(),
				Closed:    true,
			}

			// should update the lowest point
			res := adpt.TransformAndPredict(ctx, trigger, candles)
			assert.NotNil(t, res)
			d, ok := adpt.tradeInfo.Load(expert.Pair("TEST"))
			assert.True(t, ok)
			assert.Equal(t, MAX, d.(RSTradeInfo).LowPoint)
			assert.False(t, d.(RSTradeInfo).ReadyToShort)
		})
	})

	t.Run("test buy func", func(t *testing.T) {
		adpt := NewReversalScrapingStrategyV2()

		candles := []*expert.Candle{
			{
				Pair:      "TEST",
				High:      11,
				Low:       5,
				Open:      5,
				Close:     10,
				Volume:    0,
				OtherData: map[string]float64{},
				Time:      time.Now().Unix(),
				Closed:    true,
			},
			{
				Pair:      "TEST",
				High:      11,
				Low:       2,
				Open:      10,
				Close:     12,
				Volume:    0,
				OtherData: map[string]float64{},
				Time:      time.Now().Unix(),
				Closed:    true,
			},
		}

		t.Run("should update highest point", func(t *testing.T) {

			ctx := context.Background()

			trigger := expert.Candle{
				Pair:      "TEST",
				High:      11,
				Low:       2,
				Open:      12,
				Close:     10,
				Volume:    0,
				OtherData: map[string]float64{},
				Time:      time.Now().Unix(),
				Closed:    true,
			}

			// should update the lowest point
			res := adpt.TransformAndPredict(ctx, trigger, candles)
			assert.Nil(t, res)
			d, ok := adpt.tradeInfo.Load(expert.Pair("TEST"))
			assert.True(t, ok)
			assert.Equal(t, float64(12), d.(RSTradeInfo).HighPoint)
		})

		// requires first to completely update
		t.Run("should indicated ready to buy", func(t *testing.T) {
			ctx := context.Background()

			candles = append(candles, &expert.Candle{
				Pair:      "TEST",
				High:      11,
				Low:       2,
				Open:      12,
				Close:     10,
				Volume:    0,
				OtherData: map[string]float64{},
				Time:      time.Now().Unix(),
				Closed:    true,
			})

			trigger := expert.Candle{
				Pair:      "TEST",
				High:      11,
				Low:       2,
				Open:      10,
				Close:     11,
				Volume:    0,
				OtherData: map[string]float64{},
				Time:      time.Now().Unix(),
				Closed:    true,
			}

			// should update the lowest point
			res := adpt.TransformAndPredict(ctx, trigger, candles)
			assert.Nil(t, res)
			d, ok := adpt.tradeInfo.Load(expert.Pair("TEST"))
			assert.True(t, ok)
			assert.Equal(t, float64(12), d.(RSTradeInfo).HighPoint)
			assert.True(t, d.(RSTradeInfo).ReadyToBuy)
		})

		// requires first 2 to completely update
		t.Run("should buy", func(t *testing.T) {
			ctx := context.Background()

			candles = append(candles,
				&expert.Candle{
					Pair:      "TEST",
					High:      11,
					Low:       2,
					Open:      10,
					Close:     11,
					Volume:    0,
					OtherData: map[string]float64{},
					Time:      time.Now().Unix(),
					Closed:    true,
				},
				&expert.Candle{
					Pair:      "TEST",
					High:      11,
					Low:       2,
					Open:      11,
					Close:     12,
					Volume:    0,
					OtherData: map[string]float64{},
					Time:      time.Now().Unix(),
					Closed:    true,
				},
			)

			trigger := expert.Candle{
				Pair:      "TEST",
				High:      11,
				Low:       2,
				Open:      12,
				Close:     14,
				Volume:    0,
				OtherData: map[string]float64{},
				Time:      time.Now().Unix(),
				Closed:    true,
			}

			// should update the lowest point
			res := adpt.TransformAndPredict(ctx, trigger, candles)
			assert.NotNil(t, res)
			d, ok := adpt.tradeInfo.Load(expert.Pair("TEST"))
			assert.True(t, ok)
			assert.Equal(t, MIN, d.(RSTradeInfo).HighPoint)
			assert.False(t, d.(RSTradeInfo).ReadyToBuy)
		})
	})
}
