package main

import (
	"context"
	log2 "log"
	"os"
	"runtime"
	"strconv"

	"github.com/adshao/go-binance/v2"
	"go.uber.org/zap"

	TACTrader "github.com/oblessing/artisgo"
	trade "github.com/oblessing/artisgo/bot"
	"github.com/oblessing/artisgo/bot/platform"
	"github.com/oblessing/artisgo/bot/store"
	"github.com/oblessing/artisgo/bot/store/memory"
	"github.com/oblessing/artisgo/expert"
	"github.com/oblessing/artisgo/finder"
	lg "github.com/oblessing/artisgo/logger"
	"github.com/oblessing/artisgo/orders"
)

// this params would be injected
var (
	DatabaseUri = "mongodb://user:password@localhost:27017"
	logger      *log2.Logger
	logPrefix   = "app:\t"
)

func init() {
	logger = log2.New(os.Stdout, logPrefix, log2.LstdFlags|log2.Lshortfile)
} // 708, 215

func main() {
	ctx := context.Background()
	var data = os.Args[1:]
	if len(data) == 2 {
		value, err := strconv.ParseFloat(data[1], 64)
		if err != nil {
			panic(err)
		}

		TACTrader.Interval = data[0]
		TACTrader.PercentageLotSize = value
	}
	// create database
	//storage, err := mongo.NewMongoInstance(DatabaseUri)
	//if err != nil {
	//	logger.Fatal(err)
	//}

	runtime.GOMAXPROCS(6)

	// Build Expert trader
	trader := buildBinanceTrader(false, memory.NewMemoryStore(), orders.PlaceTrade, orders.Sell, trade.GetDefaultAnalysis())

	// retrieve cryptos to monitor
	supportedPairs, err := finder.GetAllUsdtPairs(ctx)
	if err != nil {
		logger.Fatal(err)
	}

	lg.Info(ctx, "about to start monitor", zap.Int("count", len(supportedPairs)))

	// create all the supported symbols
	if err = trader.WatchAndTrade(supportedPairs...); err != nil {
		logger.Fatal(err)
	}

	if err = trader.StartTrading(); err != nil {
		logger.Fatal(err)
	}
}

func buildBinanceTrader(testMode bool, storage store.Database, buyAction expert.PlaceTradeAction, sellAction expert.SellAction, defaultAnalysis []*expert.CalculateAction) trade.Trader {
	binance.UseTestnet = testMode

	return platform.NewBinanceTrader(platform.Config{
		Expert: expert.NewTrader(&expert.Config{
			Size:            6, //TODO: should be tied to strategy
			BuyAction:       buyAction,
			SellAction:      sellAction,
			Storage:         expert.NewDataSource(storage),
			DefaultAnalysis: defaultAnalysis,
		}),
	})
}
