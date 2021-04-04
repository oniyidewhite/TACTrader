package expert

import (
	"errors"
	"fmt"
	"log"
	"os"
)

// const
const (
	thresholdMin = 20
	thresholdMax = 40

	minSize = 1
	maxSize = 180

	logPrefix = "ea:\t"
)

// var
var (
	activeTrades     map[Pair]*TradeParams
	invalidSizeError = errors.New("invalid size argument")
)

// structs
type Candle struct {
	Pair Pair

	High   float64
	Low    float64
	Open   float64
	Close  float64
	Volume float64

	Time   int64
	Closed bool
}

// Struct for initiating a trade
type TradeParams struct {
	OpenTradeAt  float64
	TakeProfitAt float64
	StopLossAt   float64
	Rating       int
	Pair         Pair
}

type SellParams struct {
	IsStopLoss  bool
	SellTradeAt float64
	PL          float64
	Pair        Pair
}

type Config struct {
	Size       int
	BuyAction  BuyAction
	SellAction SellAction
	Storage    DataSource
}

// interface to save and retrieve candles
type DataSource interface {
	FetchCandles(pair Pair, size int) ([]*Candle, error)
	Persist(candle *Candle) error
}

// function for analyze the data set, returns a %value, if the trade is worth taking
type Transform func([]*Candle) *TradeParams

type BuyAction func(*TradeParams) bool
type SellAction func(*SellParams) bool

type Pair string

type system struct {
	size       int
	datasource DataSource
	buyAction  BuyAction
	sellAction SellAction
	log        *log.Logger
}

type Trader interface {
	Record(candle *Candle, transform Transform)
	TradeClosed(pair Pair)
	OnError(error)
}

// init
func init() {
	activeTrades = make(map[Pair]*TradeParams)
}

// methods
func (s *system) Record(candle *Candle, transform Transform) {
	if !candle.Closed {
		//Still actively traded
		s.tryClosing(candle)
		return
	}
	// persist the new candle
	if err := s.datasource.Persist(candle); err != nil {
		s.log.Println(err)
		return
	}

	if _, ok := activeTrades[candle.Pair]; ok {
		// we currently have an active trade
		s.tryClosing(candle)
		return
	}

	dataset, err := s.datasource.FetchCandles(candle.Pair, s.size)
	if err != nil {
		s.log.Println(err)
		return
	}

	if len(dataset) != s.size {
		s.log.Printf("invalid dataset size:%d expected:%d", len(dataset), s.size)
		return
	}

	result := transform(dataset)
	if result.Rating > thresholdMin && result.Rating < thresholdMax {
		// Call next function
		if s.buyAction(result) {
			activeTrades[candle.Pair] = result
		}
	}
}

func (s *system) TradeClosed(pair Pair) {
	delete(activeTrades, pair)
}

func (s *system) OnError(err error) {
	s.log.Printf("An error occurred: %+v", err)
}

func (s *system) tryClosing(candle *Candle) {
	params, ok := activeTrades[candle.Pair]
	if !ok {
		// we currently have an active trade
		s.log.Printf("Tried to close an already closed trade")
		return
	}

	if candle.Close <= params.StopLossAt {
		if s.sellAction(&SellParams{
			IsStopLoss:  true,
			SellTradeAt: params.TakeProfitAt,
			PL:          params.StopLossAt - params.OpenTradeAt,
			Pair:        candle.Pair,
		}) {
			s.TradeClosed(candle.Pair)
		}
	} else if candle.Close >= params.TakeProfitAt {
		fmt.Println(candle.Close, params.TakeProfitAt)
		if s.sellAction(&SellParams{
			IsStopLoss:  false,
			SellTradeAt: params.TakeProfitAt,
			PL:          params.TakeProfitAt - params.OpenTradeAt,
			Pair:        candle.Pair,
		}) {
			s.TradeClosed(candle.Pair)
		}
	}
}

// export
func NewTrader(config *Config) (Trader, error) {
	return NewTraderWithLogger(config, log.New(os.Stdout, logPrefix, log.LstdFlags|log.Lshortfile))
}

func NewTraderWithLogger(config *Config, logger *log.Logger) (Trader, error) {
	if config.Size < minSize || config.Size > maxSize {
		return nil, invalidSizeError
	}

	return &system{
		size:       config.Size,
		datasource: config.Storage,
		buyAction:  config.BuyAction,
		sellAction: config.SellAction,
		log:        logger,
	}, nil
}
