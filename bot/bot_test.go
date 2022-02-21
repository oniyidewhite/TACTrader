package bot

import (
	"context"
	"github.com/oblessing/artisgo/expert"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestScrappingDataTransform(t *testing.T) {
	t.Run("should not open trade for empty data", func(t *testing.T) {
		ctx := context.Background()

		res := ScrappingDataTransform(ctx, []*expert.Candle{})
		assert.Nil(t, res)
		assert.Equal(t, len(tradeInfo), 0)
	})

	t.Run("should track highpoint, lowpoint, readyTobuy and execute trade", func(t *testing.T) {
		ctx := context.Background()
		ctx = context.WithValue(ctx, "spread", float64(3))

		// Move up to 15
		res := ScrappingDataTransform(ctx, []*expert.Candle{
			{
				Pair:   "test",
				Closed: true,
				Open:   5,
				Close:  10,
			},
			{
				Pair:   "test",
				Closed: true,
				Open:   10,
				Close:  15,
			},
		})
		assert.Equal(t, tradeInfo["test"].HighPoint, float64(15))
		assert.False(t, tradeInfo["test"].ReadyToBuy)
		assert.Nil(t, res)
		assert.Equal(t, len(tradeInfo), 1)

		// Move down to 5
		res = ScrappingDataTransform(ctx, []*expert.Candle{
			{
				Pair:   "test",
				Closed: true,
				Open:   15,
				Close:  10,
			},
			{
				Pair:   "test",
				Closed: true,
				Open:   10,
				Close:  5,
			},
		})
		assert.Nil(t, res)
		assert.Equal(t, tradeInfo["test"].HighPoint, float64(15))
		assert.Equal(t, tradeInfo["test"].LowPoint, float64(5))
		assert.True(t, tradeInfo["test"].ReadyToBuy)
		assert.Equal(t, len(tradeInfo), 1)

		// Current update to 16
		res = ScrappingDataTransform(ctx, []*expert.Candle{
			{
				Pair:   "test",
				Closed: false,
				Open:   5,
				Close:  16,
			},
		})
		assert.NotNil(t, res)
		assert.False(t, tradeInfo["test"].ReadyToBuy)
		assert.Equal(t, res.OpenTradeAt, float64(15))

		// Move up to 20
		res = ScrappingDataTransform(ctx, []*expert.Candle{
			{
				Pair:   "test",
				Closed: true,
				Open:   5,
				Close:  16,
			},
			{
				Pair:   "test",
				Closed: true,
				Open:   16,
				Close:  20,
			},
		})
		assert.Nil(t, res)
		assert.False(t, tradeInfo["test"].ReadyToBuy)
		assert.Equal(t, tradeInfo["test"].HighPoint, float64(20))
	})
}
