package expert

import (
	"github.com/stretchr/testify/assert"
	"math/rand"
	"testing"
)

//TODO: Revamp test
type memoryStorage struct {
	store map[Pair][]*Candle
}

var recordConfig RecordConfig

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
	recordConfig = RecordConfig{}
	m.store = map[Pair][]*Candle{}
}

// TODO: Simplify test
func TestExpertSystem(t *testing.T) {
	dataSource := NewMemoryStore()
	dataSource.cleanup()

	t.Run("Transform should only be called when candle is closed, transform must contain the required no of slice", func(t *testing.T) {
		action := func(params *TradeParams) bool {
			return false
		}

		var pair Pair = "test"

		result := NewTrader(&Config{
			Size:       2,
			BuyAction:  action,
			SellAction: nil,
			Storage:    dataSource,
		})

		result.Record(NewRandomCandle(pair), func(candles []*Candle) *TradeParams {
			// We should not call this function yet
			t.FailNow()
			return &TradeParams{}
		}, recordConfig)
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

		result := NewTrader(&Config{
			Size:       2,
			BuyAction:  action,
			SellAction: nil,
			Storage:    dataSource,
		})

		result.Record(NewRandomCandle(pair), func(candles []*Candle) *TradeParams {
			// We should not call this function yet
			t.FailNow()
			return &TradeParams{}
		}, recordConfig)

		result.Record(NewRandomCandle(pair), func(candles []*Candle) *TradeParams {
			return &TradeParams{
				OpenTradeAt:  0,
				TakeProfitAt: 9,
				StopLossAt:   0,
				Rating:       22,
				Pair:         pair,
			}
		}, recordConfig)

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

		result := NewTrader(&Config{
			Size:       2,
			BuyAction:  action,
			SellAction: nil,
			Storage:    dataSource,
		})
		result.Record(NewRandomCandle(pair), func(candles []*Candle) *TradeParams {
			// We should not call this function yet
			t.FailNow()
			return &TradeParams{}
		}, recordConfig)

		result.Record(NewRandomCandle(pair), func(candles []*Candle) *TradeParams {
			return &TradeParams{
				OpenTradeAt:  0,
				TakeProfitAt: 9,
				StopLossAt:   0,
				Rating:       22,
				Pair:         pair,
			}
		}, recordConfig)

		result.TradeClosed(pair)

		result.Record(NewRandomCandle(pair), func(candles []*Candle) *TradeParams {
			return &TradeParams{
				OpenTradeAt:  0,
				TakeProfitAt: 9,
				StopLossAt:   0,
				Rating:       22,
				Pair:         pair,
			}
		}, recordConfig)
	})
	t.Run("TShould be able to open trade with only the required specs", func(t *testing.T) {
		dataSource.cleanup()
		count := 0

		recordConfig := RecordConfig{
			LotSize:        2,
			RatioToOne:     2,
			OverrideParams: true,
			TradeSize:      "100",
		}

		action := func(params *TradeParams) bool {
			assert.Equal(t, params.StopLossAt, float64(-2))
			assert.Equal(t, params.TakeProfitAt, float64(4))
			assert.Equal(t, params.TradeSize, "100")
			// open trade when called
			if count > 1 {
				t.FailNow()
			}
			count++
			return true
		}

		var pair Pair = "test"

		result := NewTrader(&Config{
			Size:       2,
			BuyAction:  action,
			SellAction: nil,
			Storage:    dataSource,
		})
		result.Record(NewRandomCandle(pair), func(candles []*Candle) *TradeParams {
			// We should not call this function yet
			t.FailNow()
			return &TradeParams{}
		}, recordConfig)

		result.Record(NewRandomCandle(pair), func(candles []*Candle) *TradeParams {
			return &TradeParams{
				OpenTradeAt:  0,
				TakeProfitAt: 9,
				StopLossAt:   0,
				Rating:       22,
				Pair:         pair,
			}
		}, recordConfig)

		result.TradeClosed(pair)

		result.Record(NewRandomCandle(pair), func(candles []*Candle) *TradeParams {
			return &TradeParams{
				OpenTradeAt:  0,
				TakeProfitAt: 9,
				StopLossAt:   0,
				Rating:       30,
				Pair:         pair,
			}
		}, recordConfig)
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

		result := NewTrader(&Config{
			Size:       1,
			BuyAction:  buyAction,
			SellAction: sellAction,
			Storage:    dataSource,
		})

		result.Record(NewRandomCandle(pair), func(candles []*Candle) *TradeParams {
			// We should not call this function yet
			return &TradeParams{
				OpenTradeAt:  1,
				TakeProfitAt: 2,
				StopLossAt:   0,
				Rating:       33,
				Pair:         "test",
			}
		}, recordConfig)

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
		}, recordConfig)
	})
	t.Run("Confirm take profit works fine", func(t *testing.T) {
		var pair Pair = "test"
		recordConfig := RecordConfig{TradeSize: "4"}

		dataSource = NewMemoryStore()
		dataSource.cleanup()
		buyAction := func(params *TradeParams) bool {
			if params.Pair != pair {
				t.Fatalf("Invalid dataset")
			}
			assert.Equal(t, params.TradeSize, "4")
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

		result := NewTrader(&Config{
			Size:       1,
			BuyAction:  buyAction,
			SellAction: sellAction,
			Storage:    dataSource,
		})

		result.Record(NewRandomCandle(pair), func(candles []*Candle) *TradeParams {
			// We should not call this function yet
			return &TradeParams{
				OpenTradeAt:  1,
				TakeProfitAt: 2,
				StopLossAt:   0,
				Rating:       33,
				Pair:         "test",
			}
		}, recordConfig)

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
		}, recordConfig)
	})
	t.Run("Test action logic to calculate some data", func(t *testing.T) {
		var pair Pair = "test"

		dataSource = NewMemoryStore()
		dataSource.cleanup()
		buyAction := func(params *TradeParams) bool {
			return true
		}

		sellAction := func(params *SellParams) bool {
			return true
		}

		result := NewTrader(&Config{
			Size:       2,
			BuyAction:  buyAction,
			SellAction: sellAction,
			Storage:    dataSource,
			DefaultAnalysis: []*CalculateAction{
				{
					Name: "k",
					Size: 2,
					Action: func(candles []*Candle) float64 {
						return 10
					},
				},
			},
		})

		result.Record(NewRandomCandle(pair), func(candles []*Candle) *TradeParams {
			// We should not call this function yet
			t.Fatalf("This function shouldn't be called")
			return &TradeParams{}
		}, recordConfig)

		result.Record(NewRandomCandle(pair), func(candles []*Candle) *TradeParams {
			return &TradeParams{}
		}, recordConfig)

		result.Record(NewRandomCandle(pair), func(candles []*Candle) *TradeParams {
			// We should not call this function yet
			if candles[0].OtherData["k"] != 10 {
				t.Fatalf("Candle should contain key:k with value:10 instead got:%f\n", candles[0].OtherData["k"])
			}
			return &TradeParams{
				OpenTradeAt:  1,
				TakeProfitAt: 2,
				StopLossAt:   0,
				Rating:       33,
				Pair:         "test",
			}
		}, recordConfig)
	})
}

func NewMemoryStore() *memoryStorage {
	return &memoryStorage{}
}

func NewRandomCandle(pair Pair) *Candle {
	return &Candle{
		Pair:      pair,
		High:      rand.Float64(),
		Low:       rand.Float64(),
		Open:      rand.Float64(),
		Close:     rand.Float64(),
		Volume:    rand.Float64(),
		Time:      rand.Int63(),
		OtherData: map[string]float64{},
		Closed:    true,
	}
}
