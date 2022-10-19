package expert

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	TACTrader "github.com/oblessing/artisgo"
)

// const
const (
	thresholdMin = 23
	thresholdMax = 40

	logPrefix = "ea:\t"
)

var (
	// TODO: Add support for placing multiple trades
	activeTrades     = sync.Map{} // map[Pair]*TradeParams{}
	invalidSizeError = errors.New("invalid size argument")
	//invalidSizeActionError = errors.New("invalid size for action argument")
)

// structs
type Candle struct {
	Pair Pair

	High   float64
	Low    float64
	Open   float64
	Close  float64
	Volume float64

	// Holds other information like RSI, MA or any other data the system needs
	OtherData map[string]float64

	Time   int64
	Closed bool
}

type TradeType string

const (
	TradeTypeLong  TradeType = "long"
	TradeTypeShort TradeType = "short"
)

// Struct for initiating a trade
type TradeParams struct {
	TradeType    TradeType `json:"trade_type"`
	OpenTradeAt  float64   `json:"open_trade_at"`
	OrderID      int64     `json:"order_id"`
	TakeProfitAt float64   `json:"take_profit_at"`
	StopLossAt   float64   `json:"stop_loss_at"`
	TradeSize    string    `json:"trade_size"`
	Rating       int       `json:"rating"`
	Pair         Pair      `json:"pair"`
	CreatedAt    time.Time `json:"time"`
}

type SellParams struct {
	IsStopLoss  bool
	SellTradeAt float64
	PL          float64
	OrderID     int64
	TradeSize   string
	Pair        Pair
	TradeType   TradeType `json:"trade_type"`
}

type Config struct {
	Size            int
	BuyAction       PlaceTradeAction
	SellAction      SellAction
	Storage         DataSource
	DefaultAnalysis []*CalculateAction
}

// interface to save and retrieve candles
type DataSource interface {
	FetchCandles(ctx context.Context, pair Pair, size int) ([]*Candle, error)
	Persist(ctx context.Context, candle *Candle) error
}

// Transform for analyze the data set, returns a %value, if the trade is worth taking
type Transform func(ctx context.Context, trigger Candle, candles []*Candle) *TradeParams

type PlaceTradeAction func(*TradeParams) bool
type SellAction func(*SellParams) bool

type Pair string

type CalculateAction struct {
	Name   string
	Size   int8 // This is ignored
	Action func([]*Candle) float64
}

type system struct {
	size            int
	datasource      DataSource
	openTradeAction PlaceTradeAction
	sellAction      SellAction
	log             *log.Logger
	calculateAction []*CalculateAction
}

type RecordConfig struct {
	Spread float64
	// Represents the percentage change
	LotSize    float64
	RatioToOne float64
	// Override expert stop & take profit with config info
	OverrideParams bool
	// Represents the trade size
	TradeSize string
}

type Trader interface {
	Record(ctx context.Context, candle *Candle, transform Transform, config RecordConfig)
	TradeClosed(ctx context.Context, pair Pair)
	OnError(context.Context, error)
}

// IsUp
func (c *Candle) IsUp() bool {
	return c.Close > c.Open
}

func (s *system) Record(ctx context.Context, c *Candle, transform Transform, config RecordConfig) {
	//Try checking if we need to close any trade,
	//do not use heikin ashi to close trade.
	s.tryClosing(ctx, c)

	// Check if the time is still running,
	//return, so we do not persist it.
	if !c.Closed {
		return
	}

	// Convert card to heikin ashi.
	candles, _ := s.datasource.FetchCandles(ctx, c.Pair, s.size)
	var previousCandle = c
	if len(candles) != 0 {
		previousCandle = candles[0]
	}
	candle := convertToHeikinAshi(previousCandle, c)

	// apply actions; MA, RSI, etc
	for _, action := range s.calculateAction {
		if len(candles) != s.size {
			break
		}

		candle.OtherData[action.Name] = action.Action(append([]*Candle{candle}, candles...))
	}

	// persist the new candle
	if err := s.datasource.Persist(ctx, candle); err != nil {
		s.log.Printf("Error saving record :%+v", err)
		return
	}

	// Check if we have open trade.
	// TODO: support opening of multiple positions.
	if _, ok := read(candle.Pair); ok {
		return
	}

	dataset, err := s.datasource.FetchCandles(ctx, candle.Pair, s.size)
	if err != nil {
		s.log.Printf("Error fetching record :%+v", err)
		return
	}

	if len(dataset) < 2 {
		return
	}

	s.processTrade(ctx, *c, transform, config, dataset)
}

func (s *system) processTrade(ctx context.Context, c Candle, transform Transform, config RecordConfig, dataset []*Candle) {
	ctx = context.WithValue(ctx, "spread", config.Spread)
	result := transform(ctx, c, dataset)
	// Check if we can act on the data
	if result == nil {
		return
	}
	result.TradeSize = config.TradeSize

	var lotSize = config.LotSize * result.OpenTradeAt
	var tradeSize = (1 / result.OpenTradeAt) * TACTrader.TradeAmount
	switch result.TradeType {
	case TradeTypeLong:
		stopLoss := result.OpenTradeAt - lotSize
		takeProfit := result.OpenTradeAt + (lotSize * config.RatioToOne)
		result.TakeProfitAt = takeProfit
		result.StopLossAt = stopLoss
		result.TradeSize = fmt.Sprintf("%f", tradeSize)
	case TradeTypeShort:
		stopLoss := result.OpenTradeAt + lotSize
		takeProfit := result.OpenTradeAt - (lotSize * config.RatioToOne)
		result.TakeProfitAt = takeProfit
		result.StopLossAt = stopLoss
		result.TradeSize = fmt.Sprintf("%f", tradeSize)
	}

	if s.openTradeAction(result) {
		write(result.Pair, result)
	}
}

func convertToHeikinAshi(older *Candle, newer *Candle) *Candle {
	if newer == nil {
		return nil
	}
	if older == nil {
		older = newer
	}

	// close
	newClose := 0.25 * (newer.Open + newer.Close + newer.High + newer.Low)
	newOpen := 0.5 * (older.Open + older.Close)
	newLow := lowest([]float64{newer.Low, newOpen, newClose})
	newHigh := highest([]float64{newer.High, newOpen, newClose})

	result := &Candle{
		Pair:      newer.Pair,
		High:      newHigh,
		Low:       newLow,
		Open:      newOpen,
		Close:     newClose,
		Volume:    newer.Volume,
		OtherData: newer.OtherData,
		Time:      newer.Time,
		Closed:    newer.Closed,
	}

	return result
}

func lowest(arr []float64) float64 {
	value := arr[0]
	for _, v := range arr {
		if v < value {
			value = v
		}
	}
	return value
}

func highest(arr []float64) float64 {
	value := arr[0]
	for _, v := range arr {
		if v > value {
			value = v
		}
	}
	return value
}

func (s *system) TradeClosed(ctx context.Context, pair Pair) {
	remove(pair)
}

func (s *system) OnError(ctx context.Context, err error) {
	s.log.Printf("An error occurred: %+v\n", err)
}

func (s *system) tryClosing(ctx context.Context, candle *Candle) {
	params, ok := read(candle.Pair)
	if !ok {
		// we currently don't have an active trade
		return
	}

	// try closing based on trade type.
	switch params.TradeType {
	case TradeTypeLong:
		if candle.Close >= params.TakeProfitAt {
			if s.sellAction(&SellParams{
				IsStopLoss:  false,
				SellTradeAt: candle.Close,
				PL:          candle.Close - params.OpenTradeAt,
				Pair:        candle.Pair,
				TradeSize:   params.TradeSize,
				OrderID:     params.OrderID,
				TradeType:   params.TradeType,
			}) {
				s.TradeClosed(ctx, candle.Pair)
			}
		} else if candle.Close <= params.StopLossAt {
			if s.sellAction(&SellParams{
				IsStopLoss:  true,
				SellTradeAt: candle.Close,
				PL:          candle.Close - params.OpenTradeAt,
				Pair:        candle.Pair,
				TradeSize:   params.TradeSize,
				OrderID:     params.OrderID,
				TradeType:   params.TradeType,
			}) {
				s.TradeClosed(ctx, candle.Pair)
			}
		}
	case TradeTypeShort:
		if candle.Close <= params.TakeProfitAt {
			if s.sellAction(&SellParams{
				IsStopLoss:  false,
				SellTradeAt: candle.Close,
				PL:          params.OpenTradeAt - candle.Close,
				Pair:        candle.Pair,
				TradeSize:   params.TradeSize,
				OrderID:     params.OrderID,
				TradeType:   params.TradeType,
			}) {
				s.TradeClosed(ctx, candle.Pair)
			}
		} else if candle.Close >= params.StopLossAt {
			if s.sellAction(&SellParams{
				IsStopLoss:  true,
				SellTradeAt: candle.Close,
				PL:          params.OpenTradeAt - candle.Close,
				Pair:        candle.Pair,
				TradeSize:   params.TradeSize,
				OrderID:     params.OrderID,
				TradeType:   params.TradeType,
			}) {
				s.TradeClosed(ctx, candle.Pair)
			}
		}
	}
}

func read(key Pair) (*TradeParams, bool) {
	result, ok := activeTrades.Load(key)
	if !ok {
		return nil, false
	}

	return result.(*TradeParams), ok
}

func remove(key Pair) {
	activeTrades.Delete(key)
}

func write(key Pair, data *TradeParams) {
	activeTrades.Store(key, data)
}

// NewTrader returns a new Trader with logger enabled
func NewTrader(config *Config) Trader {
	return NewTraderWithLogger(config, log.New(os.Stdout, logPrefix, log.LstdFlags|log.Lshortfile))
}

func NewTraderWithLogger(config *Config, logger *log.Logger) Trader {
	return &system{
		size:            config.Size,
		datasource:      config.Storage,
		openTradeAction: config.BuyAction,
		sellAction:      config.SellAction,
		log:             logger,
		calculateAction: config.DefaultAnalysis,
	}
}
