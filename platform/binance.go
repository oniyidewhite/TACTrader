package platform

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/adshao/go-binance/v2"
	"go.uber.org/zap"

	settings "github.com/oblessing/artisgo"

	"github.com/oblessing/artisgo/bot"
	"github.com/oblessing/artisgo/expert"
	"github.com/oblessing/artisgo/logger"
)

// myBinance represent My Binance API configuration
type myBinance struct {
	config settings.Config
	trader expert.Trader
}

type TradingService interface {
	StartTrading(ctx context.Context, pairs ...bot.PairConfig) error
}

func (r *myBinance) StartTrading(ctx context.Context, pairs ...bot.PairConfig) error {
	logger.Info(ctx, "service is starting up")

	wg := sync.WaitGroup{}
	wg.Add(len(pairs))

	errHandler := func(err error) {
		logger.Error(ctx, "start_trading: an error occurred", zap.Error(err))
	}

	// Start all the current pairs
	for _, p := range pairs {
		p := p
		go func() {
			wsKlineHandler := func(event *binance.WsKlineEvent) {
				ctx := context.Background()

				r.trader.Record(ctx, convert(event), p.Strategy, expert.RecordConfig{
					LotSize:         p.LotSize,
					RatioToOne:      p.RatioToOne,
					CandleSize:      p.CandleSize,
					DefaultAnalysis: p.DefaultAnalysis,
				})
			}

			// We restart if we encounter an error.
			var hasStarted = false
			for {
				logger.Info(ctx, "starting watcher", zap.String("pair", p.Pair), zap.String("period", p.Period))
				doneC, _, err := binance.WsKlineServe(p.Pair, p.Period, wsKlineHandler, errHandler)
				if err != nil {
					logger.Error(ctx, "start_trading: an error occurred, will resume after 30s", zap.Error(err))
					<-time.After(30 * time.Second)
					continue
				}
				if !hasStarted {
					wg.Done()
					hasStarted = true
				}
				<-doneC
			}
		}()
	}

	// Lock
	wg.Wait()
	logger.Info(ctx, "service is running")
	<-make(chan struct{})
	return nil
}

// check if we can close this trade.
// if trade doesn't exist we still return false
func convert(kline *binance.WsKlineEvent) *expert.Candle {
	high, err := parseString(kline.Kline.High)
	if err != nil {
		return nil
	}
	low, err := parseString(kline.Kline.Low)
	if err != nil {
		return nil
	}
	open, err := parseString(kline.Kline.Open)
	if err != nil {
		return nil
	}
	cl, err := parseString(kline.Kline.Close)
	if err != nil {
		return nil
	}
	vol, err := parseString(kline.Kline.Volume)
	if err != nil {
		return nil
	}

	return &expert.Candle{
		Pair:      expert.Pair(kline.Symbol),
		High:      high,
		Low:       low,
		Open:      open,
		Close:     cl,
		Volume:    vol,
		Time:      kline.Time,
		Closed:    kline.Kline.IsFinal,
		OtherData: map[string]float64{},
	}
}
func parseString(value string) (float64, error) {
	return strconv.ParseFloat(value, 64)
}

// NewSymbolDatasource allows us to monitor and receive update during price changes.
func NewSymbolDatasource(config settings.Config, trader expert.Trader) TradingService {
	binance.UseTestnet = config.IsTestMode()

	return &myBinance{
		trader: trader,
		config: config,
	}
}
