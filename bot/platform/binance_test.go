package platform

import (
	"github.com/adshao/go-binance/v2"
	"github.com/oblessing/artisgo/bot"
	"github.com/oblessing/artisgo/expert"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	t.Run("should return error if nil", func(t *testing.T) {
		got := NewBinanceTrader(Config{
			Client: nil,
			Expert: nil,
		})
		assert.NotNil(t, got)
	})
}

func Test_convert(t *testing.T) {
	t.Run("should convert chart data correctly", func(t *testing.T) {
		start := time.Now().Unix()
		end := time.Now().Add(1 * time.Minute).Unix()
		got := convert(&binance.WsKlineEvent{
			Event:  "event",
			Time:   end,
			Symbol: "test",
			Kline: binance.WsKline{
				StartTime:            start,
				EndTime:              end,
				Symbol:               "test",
				Interval:             "1m",
				FirstTradeID:         1,
				LastTradeID:          2,
				Open:                 "1.0",
				Close:                "1.22",
				High:                 "1.99",
				Low:                  "0.99",
				Volume:               "300",
				TradeNum:             3,
				IsFinal:              true,
				QuoteVolume:          "3001",
				ActiveBuyVolume:      "30",
				ActiveBuyQuoteVolume: "3",
			},
		})
		assert.Equal(t, &expert.Candle{
			Pair:      "test",
			High:      1.99,
			Low:       0.99,
			Open:      1.0,
			Close:     1.22,
			Volume:    300,
			OtherData: map[string]float64{},
			Time:      end,
			Closed:    true,
		}, got)
	})
}

func Test_myBinance_WatchAndTrade(t *testing.T) {
	t.Run("should add record to watch list", func(t *testing.T) {
		r := &myBinance{}
		err := r.WatchAndTrade(bot.PairConfig{
			Pair:            "t",
			Period:          "1m",
			TradeSize:       "3",
			Strategy:        nil,
			DisableStopLoss: false,
		}, bot.PairConfig{
			Pair:            "t2",
			Period:          "1m",
			TradeSize:       "3",
			Strategy:        nil,
			DisableStopLoss: false,
		})
		assert.NoError(t, err)
		assert.Equal(t, 2, len(r.pairs))
	})

	t.Run("should fail if bot already started", func(t *testing.T) {
		r := &myBinance{}
		r.hasStarted = true

		err := r.WatchAndTrade(bot.PairConfig{
			Pair:            "t",
			Period:          "1m",
			TradeSize:       "3",
			Strategy:        nil,
			DisableStopLoss: false,
		}, bot.PairConfig{
			Pair:            "t2",
			Period:          "1m",
			TradeSize:       "3",
			Strategy:        nil,
			DisableStopLoss: false,
		})
		assert.Error(t, err)
		assert.Equal(t, 0, len(r.pairs))
	})
}
