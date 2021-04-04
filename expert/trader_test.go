package expert

import (
	"math/rand"
	"testing"
)

type memoryStorage struct {
	store map[Pair][]*Candle
}

func (m *memoryStorage) FetchCandles(pair Pair, size int) ([]*Candle, error) {
	c := m.store[pair]
	start := len(c) - size
	if start < 0 {
		return nil, invalidSizeError
	}
	return c[start:], nil
}

func (m *memoryStorage) Persist(candle *Candle) error {
	m.store[candle.Pair] = append(m.store[candle.Pair], candle)
	return nil
}

func (m *memoryStorage) cleanup() {
	m.store = map[Pair][]*Candle{}
}

// TODO: Simplify test
func TestExpertSystem(t *testing.T) {
	dataSource := NewMemoryStore()
	dataSource.cleanup()

	t.Run("Should throw error if size is 0", func(t *testing.T) {
		action := func(params *TradeParams) bool {
			return false
		}
		_, err := NewTrader(&Config{
			Size:       0,
			BuyAction:  action,
			SellAction: nil,
			Storage:    nil,
		})
		if err == nil {
			t.FailNow()
		}
	})
	t.Run("Should create new instance of system if params are valid", func(t *testing.T) {
		action := func(params *TradeParams) bool {
			return false
		}
		result, err := NewTrader(&Config{
			Size:       2,
			BuyAction:  action,
			SellAction: nil,
			Storage:    nil,
		})
		if err != nil {
			t.FailNow()
		}

		if result == nil {
			t.FailNow()
		}
	})
	t.Run("Transform should only be called when candle is closed, transform must contain the required no of slice", func(t *testing.T) {
		action := func(params *TradeParams) bool {
			return false
		}

		var pair Pair = "test"

		result, err := NewTrader(&Config{
			Size:       2,
			BuyAction:  action,
			SellAction: nil,
			Storage:    dataSource,
		})
		if err != nil {
			t.FailNow()
		}

		result.Record(NewRandomCandle(pair), func(candles []*Candle) *TradeParams {
			// We should not call this function yet
			t.FailNow()
			return &TradeParams{}
		})
	})
	t.Run("Transform should not open any new trade until current one is closed", func(t *testing.T) {
		dataSource.cleanup()
		count := 0
		action := func(params *TradeParams) bool {
			// open trade when called
			if count > 0 {
				t.FailNow()
			}
			count++
			return true
		}

		var pair Pair = "test"

		result, err := NewTrader(&Config{
			Size:       2,
			BuyAction:  action,
			SellAction: nil,
			Storage:    dataSource,
		})
		if err != nil {
			t.FailNow()
		}

		result.Record(NewRandomCandle(pair), func(candles []*Candle) *TradeParams {
			// We should not call this function yet
			t.FailNow()
			return &TradeParams{}
		})

		result.Record(NewRandomCandle(pair), func(candles []*Candle) *TradeParams {
			return &TradeParams{
				OpenTradeAt:  0,
				TakeProfitAt: 9,
				StopLossAt:   0,
				Rating:       22,
				Pair:         pair,
			}
		})

	})
	t.Run("Transform should open a new trade after current one is closed", func(t *testing.T) {
		dataSource.cleanup()
		count := 0
		action := func(params *TradeParams) bool {
			// open trade when called
			if count > 1 {
				t.FailNow()
			}
			count++
			return true
		}

		var pair Pair = "test"

		result, err := NewTrader(&Config{
			Size:       2,
			BuyAction:  action,
			SellAction: nil,
			Storage:    dataSource,
		})
		if err != nil {
			t.FailNow()
		}

		result.Record(NewRandomCandle(pair), func(candles []*Candle) *TradeParams {
			// We should not call this function yet
			t.FailNow()
			return &TradeParams{}
		})

		result.Record(NewRandomCandle(pair), func(candles []*Candle) *TradeParams {
			return &TradeParams{
				OpenTradeAt:  0,
				TakeProfitAt: 9,
				StopLossAt:   0,
				Rating:       22,
				Pair:         pair,
			}
		})

		result.TradeClosed(pair)

		result.Record(NewRandomCandle(pair), func(candles []*Candle) *TradeParams {
			return &TradeParams{
				OpenTradeAt:  0,
				TakeProfitAt: 9,
				StopLossAt:   0,
				Rating:       22,
				Pair:         pair,
			}
		})
	})
	t.Run("Confirm stop loss works fine", func(t *testing.T) {
		var pair Pair = "test"

		dataSource = NewMemoryStore()
		dataSource.cleanup()
		buyAction := func(params *TradeParams) bool {
			if params.Pair != pair {
				t.Fatalf("Invalid dataset")
			}
			return true
		}

		sellAction := func(params *SellParams) bool {
			if params.Pair != pair {
				t.Fatalf("Invalid dataset")
			}
			if !params.IsStopLoss {
				t.Fatalf("This should be a stoploss: %+v", params)
			}
			return true
		}

		result, err := NewTrader(&Config{
			Size:       1,
			BuyAction:  buyAction,
			SellAction: sellAction,
			Storage:    dataSource,
		})
		if err != nil {
			t.FailNow()
		}

		result.Record(NewRandomCandle(pair), func(candles []*Candle) *TradeParams {
			// We should not call this function yet
			return &TradeParams{
				OpenTradeAt:  1,
				TakeProfitAt: 2,
				StopLossAt:   0,
				Rating:       33,
				Pair:         "test",
			}
		})

		result.Record(&Candle{
			Pair:   "test",
			High:   0,
			Low:    0,
			Open:   0,
			Close:  0,
			Volume: 0,
			Time:   0,
			Closed: true,
		}, func(candles []*Candle) *TradeParams {
			// We should not call this function yet
			return &TradeParams{
				OpenTradeAt:  candles[0].Open,
				TakeProfitAt: candles[0].Close,
				StopLossAt:   candles[0].Open,
				Rating:       33,
				Pair:         "test",
			}
		})
	})
	t.Run("Confirm take profit works fine", func(t *testing.T) {
		var pair Pair = "test"

		dataSource = NewMemoryStore()
		dataSource.cleanup()
		buyAction := func(params *TradeParams) bool {
			if params.Pair != pair {
				t.Fatalf("Invalid dataset")
			}
			return true
		}

		sellAction := func(params *SellParams) bool {
			if params.Pair != pair {
				t.Fatalf("Invalid dataset:%+v", params)
			}
			if params.IsStopLoss {
				t.Fatalf("This should be a take profit:%+v", params)
			}
			return true
		}

		result, err := NewTrader(&Config{
			Size:       1,
			BuyAction:  buyAction,
			SellAction: sellAction,
			Storage:    dataSource,
		})
		if err != nil {
			t.FailNow()
		}

		result.Record(NewRandomCandle(pair), func(candles []*Candle) *TradeParams {
			// We should not call this function yet
			return &TradeParams{
				OpenTradeAt:  1,
				TakeProfitAt: 2,
				StopLossAt:   0,
				Rating:       33,
				Pair:         "test",
			}
		})

		result.Record(&Candle{
			Pair:   "test",
			High:   0,
			Low:    0,
			Open:   0,
			Close:  3,
			Volume: 0,
			Time:   0,
			Closed: true,
		}, func(candles []*Candle) *TradeParams {
			// We should not call this function yet
			return &TradeParams{
				OpenTradeAt:  candles[0].Open,
				TakeProfitAt: candles[0].Close,
				StopLossAt:   candles[0].Open,
				Rating:       33,
				Pair:         "test",
			}
		})
	})
}

func NewMemoryStore() *memoryStorage {
	return &memoryStorage{}
}

func NewRandomCandle(pair Pair) *Candle {
	return &Candle{
		Pair:   pair,
		High:   rand.Float64(),
		Low:    rand.Float64(),
		Open:   rand.Float64(),
		Close:  rand.Float64(),
		Volume: rand.Float64(),
		Time:   rand.Int63(),
		Closed: true,
	}
}
