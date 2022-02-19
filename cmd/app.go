package main

import (
	"github.com/adshao/go-binance/v2"
	trade "github.com/oblessing/artisgo/bot"
	"github.com/oblessing/artisgo/bot/platform"
	"github.com/oblessing/artisgo/bot/store"
	"github.com/oblessing/artisgo/bot/store/mongo"
	"github.com/oblessing/artisgo/expert"
	"github.com/oblessing/artisgo/orders"
	log2 "log"
	"os"
)

// this params would be injected
var (
	DatabaseUri = "mongodb://user:password@localhost:27017"
	logger      *log2.Logger
	logPrefix   = "app:\t"
)

func init() {
	logger = log2.New(os.Stdout, logPrefix, log2.LstdFlags|log2.Lshortfile)
}

func main() {
	// create database
	storage, err := mongo.NewMongoInstance(DatabaseUri)
	if err != nil {
		logger.Fatal(err)
	}

	// Build Expert trader
	trader := buildBinanceTrader(false, storage, orders.Buy, orders.Sell, trade.GetDefaultAnalysis())

	// create all the supported symbols
	if err = trader.WatchAndTrade(orders.TradeList...); err != nil {
		logger.Fatal(err)
	}

	if err = trader.StartTrading(); err != nil {
		logger.Fatal(err)
	}
}

func buildBinanceTrader(testMode bool, storage store.Database, buyAction expert.BuyAction, sellAction expert.SellAction, defaultAnalysis []*expert.CalculateAction) trade.Trader {
	binance.UseTestnet = testMode

	return platform.NewBinanceTrader(platform.Config{
		Expert: expert.NewTrader(&expert.Config{
			Size:            18, //TODO: should be tied to strategy
			BuyAction:       buyAction,
			SellAction:      sellAction,
			Storage:         expert.NewDataSource(storage),
			DefaultAnalysis: defaultAnalysis,
		}),
	})
}
