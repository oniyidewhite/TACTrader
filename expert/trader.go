package expert

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	settings "github.com/oblessing/artisgo"
	"github.com/oblessing/artisgo/logger"
	"github.com/oblessing/artisgo/store"
)

const (
	TradeTypeLong  TradeType = "long"
	TradeTypeShort TradeType = "short"
)

var (
	// TODO: Add support for placing multiple trades for a specific symbol
	activeTrades = sync.Map{} // map[Pair]*TradeParams{}
)

type TradeType string

// Transform for analyze the data set, returns a %value, if the trade is worth taking
type Transform func(ctx context.Context, trigger Candle, candles []*Candle) *TradeParams

type Pair string

type Candle struct {
	Pair   Pair
	High   float64
	Low    float64
	Open   float64
	Close  float64
	Volume float64
	// Holds other information like RSI, MA or any other data the system needs
	OtherData map[string]float64
	Time      int64
	Closed    bool
}

func (c *Candle) IsUp() bool {
	return c.Close > c.Open
}

// TradeParams for initiating a trade
type TradeParams struct {
	TradeType         TradeType          `json:"trade_type"`
	OriginalTradeType TradeType          `json:"original_trade_type"`
	OpenTradeAt       string             `json:"open_trade_at"`
	Volume            float64            `json:"volume"`
	OrderID           string             `json:"order_id"`
	TakeProfitAt      string             `json:"take_profit_at"`
	StopLossAt        string             `json:"stop_loss_at"`
	TradeSize         string             `json:"trade_size"`
	Rating            int                `json:"rating"` // Deprecated
	Pair              Pair               `json:"pair"`
	CreatedAt         time.Time          `json:"time"`
	Attribs           map[string]float64 `json:"others"`
	CanNotOverride    bool               `json:"canNotOverride"`
}

func (t TradeParams) OpenTradeAtV() float64 {
	r, _ := strconv.ParseFloat(t.OpenTradeAt, 64)
	return r
}
func (t TradeParams) TakeProfitAtV() float64 {
	r, _ := strconv.ParseFloat(t.TakeProfitAt, 64)
	return r
}
func (t TradeParams) StopLossAtV() float64 {
	r, _ := strconv.ParseFloat(t.StopLossAt, 64)
	return r
}

type TradeData struct {
	OrderID       string
	ClientOrderID string
}

type SellParams struct {
	IsStopLoss  bool
	SellTradeAt float64
	PL          float64
	OrderID     string
	TradeSize   string
	Pair        Pair
	TradeType   TradeType `json:"trade_type"`
}

type CalculateAction struct {
	Name   string
	Size   int8 // This is ignored
	Action func([]*Candle) float64
}

type system struct {
	settings     settings.Config
	datasource   DataSource
	orderService OrderService
}

type RecordConfig struct {
	// Represents the percentage change
	LotSize         float64
	RatioToOne      float64
	CandleSize      int
	AdditionalData  []string // minPrice, stepSize, precision
	DefaultAnalysis []*CalculateAction
}

type DataSource interface {
	FetchCandles(ctx context.Context, pair Pair, size int) ([]*Candle, error)
	Persist(ctx context.Context, candle *Candle) error
}

type OrderService interface {
	PlaceTrade(ctx context.Context, params TradeParams) (TradeData, error)
	CloseTrade(ctx context.Context, params SellParams) (bool, error)
}

type Trader interface {
	Record(ctx context.Context, candle *Candle, transform Transform, config RecordConfig)
}

func NewExpertTrader(config settings.Config, storage store.Database, service OrderService) *system {
	return &system{
		settings:     config,
		datasource:   NewDataSource(storage),
		orderService: service,
	}
}

func (s *system) Record(ctx context.Context, c *Candle, transform Transform, config RecordConfig) {
	// Try checking if we need to close any trade,
	// do not use heikin ashi to close trade.
	s.tryClosing(ctx, c)

	// Check if the time is still running,
	// return, so we do not persist it.
	if !c.Closed {
		return
	}

	// Convert card to heikin ashi.
	candles, _ := s.datasource.FetchCandles(ctx, c.Pair, config.CandleSize)
	var previousCandle = c
	if len(candles) != 0 {
		previousCandle = candles[len(candles)-1]
	}
	candle := convertToHeikinAshi(previousCandle, c)

	// apply actions; MA, RSI, etc
	for _, action := range config.DefaultAnalysis {
		if len(candles) != config.CandleSize {
			break
		}

		candle.OtherData[action.Name] = action.Action(append([]*Candle{candle}, candles...))
	}

	// persist the new candle
	if err := s.datasource.Persist(ctx, candle); err != nil {
		logger.Error(ctx, "error persisting record", zap.Error(err))
		return
	}

	if len(candles) < 2 {
		return
	}

	s.processTrade(ctx, *c, transform, config, candles)
}

func (s *system) processTrade(ctx context.Context, c Candle, transform Transform, config RecordConfig, dataset []*Candle) {
	lotPrecision := findNumberOfDecimal(config.AdditionalData[1])
	quotePrecision := findNumberOfDecimal(config.AdditionalData[0])

	result := transform(ctx, c, dataset)
	// Check if we can act on the data
	if result == nil {
		return
	}

	// trading amount is
	var tradeSize = ((1 / result.OpenTradeAtV()) * s.settings.TradeAmount) * config.LotSize
	var buyPrice = fmt.Sprintf("%v", RoundToDecimalPoint(result.OpenTradeAtV(), quotePrecision))

	// use RSI to deduce the direction, extreme high(short) extreme low(long)
	// // 82, 93 || 27
	rsi, ok := c.OtherData["RSI"]
	if !ok {
		return
	}

	// t := time.Now().UTC()
	// if t.Weekday() == time.Monday ||
	// 	t.UTC().Weekday() == time.Friday ||
	// 	t.UTC().Weekday() == time.Saturday ||
	// 	t.UTC().Weekday() == time.Sunday {
	// 	return
	// }
	// // US market							// HongKong/beijing market
	// if !(t.Hour() >= 13 && t.Hour() <= 20) || !(t.Hour() >= 4 && t.Hour() <= 11) {
	// 	return
	// }
	//
	// if t.Before(settings.StartTime) {
	// 	return
	// }

	result.OriginalTradeType = result.TradeType
	result.Attribs = c.OtherData

	if rsi > 81 && result.TradeType == TradeTypeShort {
		result.TradeType = TradeTypeShort
	} else if rsi < 23 && result.TradeType == TradeTypeLong {
		result.TradeType = TradeTypeLong
	} else {
		logger.Info(ctx, "did not take", zap.Any("ignored", result))
		return
	}

	switch result.TradeType {
	case TradeTypeLong:
		stopLoss := result.OpenTradeAtV() - ((result.OpenTradeAtV()) / config.LotSize)
		// since leverage is 10 times
		// current price + ((current price * ratio) / 10)
		var takeProfit = result.OpenTradeAtV() + ((result.OpenTradeAtV() * config.RatioToOne) / config.LotSize)
		result.TakeProfitAt = fmt.Sprintf("%v", RoundToDecimalPoint(takeProfit, quotePrecision))
		result.StopLossAt = fmt.Sprintf("%v", RoundToDecimalPoint(stopLoss, quotePrecision))
		result.TradeSize = fmt.Sprintf("%v", RoundToDecimalPoint(tradeSize, lotPrecision))
	case TradeTypeShort:
		stopLoss := result.OpenTradeAtV() + ((result.OpenTradeAtV()) / config.LotSize)
		// since leverage is 10 times
		// current price + ((current price * ratio) / 10)
		var takeProfit = result.OpenTradeAtV() - ((result.OpenTradeAtV() * config.RatioToOne) / config.LotSize)
		result.TakeProfitAt = fmt.Sprintf("%v", RoundToDecimalPoint(takeProfit, quotePrecision))
		result.StopLossAt = fmt.Sprintf("%v", RoundToDecimalPoint(stopLoss, quotePrecision))
		result.TradeSize = fmt.Sprintf("%v", RoundToDecimalPoint(tradeSize, lotPrecision))
	}
	// set timestamp
	result.CreatedAt = time.Now().UTC()
	// Set additional attribs for logging
	result.Attribs = c.OtherData
	result.OpenTradeAt = buyPrice
	result.Volume = c.Volume

	// TODO(oblessing): don't allow close at the same price, throw error so moderator can close it.
	if result.TakeProfitAt == result.OpenTradeAt || result.StopLossAt == result.OpenTradeAt {
		m := "same open + stop"
		if result.TakeProfitAt == result.OpenTradeAt {
			m = "same open + profit"
		}
		logger.Error(ctx, "trade mismatch", zap.String("mismatch", m), zap.Any("t", result))
		return
	}

	if result != nil {
		// Check if we have open trade.
		// TODO: support opening of multiple positions.
		if _, ok := read(result.Pair); ok {
			logger.Error(ctx, "already have an open trade", zap.Any("ignored", result))
			return
		}

		trd, err := s.orderService.PlaceTrade(ctx, *result)
		if err != nil {
			return
		}

		result.OrderID = trd.OrderID

		write(result.Pair, result)
	}
}

func findNumberOfDecimal(v string) uint8 {
	data := strings.Split(v, ".")
	if len(data) == 2 {
		var result uint8 = 0
		for _, i := range data[1] {
			if i == '0' {
				result += 1
			} else {
				return result + 1
			}
		}
	}

	return 0
}

// RoundToDecimalPoint take an amount then rounds it to the upper 2 decimal point if the value is more than 2 decimal point.
func RoundToDecimalPoint(amount float64, precision uint8) float64 {
	amountString := fmt.Sprintf("%.8f", amount)

	amountSplit := strings.Split(amountString, ".")

	if len(amountSplit) != 2 {
		return amount
	}

	if len(amountSplit[1]) <= int(precision) {
		return amount
	}

	valueAmount := fmt.Sprintf("%s%s", amountSplit[0], amountSplit[1][:int(precision)])
	result, _ := strconv.ParseInt(valueAmount, 10, 64)

	return float64(result) / math.Pow(float64(10), float64(precision))
}

func (s *system) tradeClosed(pair Pair) {
	remove(pair)
}

func (s *system) tryClosing(ctx context.Context, candle *Candle) {
	params, ok := read(candle.Pair)
	if !ok {
		// we currently don't have an active trade
		return
	}

	var err error
	var closedTrade bool

	// try closing based on trade type.
	switch params.TradeType {
	case TradeTypeLong:
		if candle.Close >= params.TakeProfitAtV() {
			closedTrade, err = s.orderService.CloseTrade(ctx, SellParams{
				IsStopLoss:  false,
				SellTradeAt: candle.Close,
				PL:          candle.Close - params.OpenTradeAtV(),
				Pair:        candle.Pair,
				TradeSize:   params.TradeSize,
				OrderID:     params.OrderID,
				TradeType:   params.TradeType,
			})
		} else if candle.Close <= params.StopLossAtV() {
			closedTrade, err = s.orderService.CloseTrade(ctx, SellParams{
				IsStopLoss:  true,
				SellTradeAt: candle.Close,
				PL:          candle.Close - params.OpenTradeAtV(),
				Pair:        candle.Pair,
				TradeSize:   params.TradeSize,
				OrderID:     params.OrderID,
				TradeType:   params.TradeType,
			})
		}
	case TradeTypeShort:
		if candle.Close <= params.TakeProfitAtV() {
			closedTrade, err = s.orderService.CloseTrade(ctx, SellParams{
				IsStopLoss:  false,
				SellTradeAt: candle.Close,
				PL:          params.OpenTradeAtV() - candle.Close,
				Pair:        candle.Pair,
				TradeSize:   params.TradeSize,
				OrderID:     params.OrderID,
				TradeType:   params.TradeType,
			})
		} else if candle.Close >= params.StopLossAtV() {
			closedTrade, err = s.orderService.CloseTrade(ctx, SellParams{
				IsStopLoss:  true,
				SellTradeAt: candle.Close,
				PL:          params.OpenTradeAtV() - candle.Close,
				Pair:        candle.Pair,
				TradeSize:   params.TradeSize,
				OrderID:     params.OrderID,
				TradeType:   params.TradeType,
			})
		}
	}

	if err != nil {
		logger.Error(ctx, "ea_trader: error occurred while attempting to close trade", zap.Error(err), zap.Any("p", params))
		return
	}

	if closedTrade {
		s.tradeClosed(candle.Pair)
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
