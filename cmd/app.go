package main

import (
	"context"
	log2 "log"
	"os"
	"runtime"

	"go.uber.org/zap"

	settings "github.com/oblessing/artisgo"
	"github.com/oblessing/artisgo/expert"
	"github.com/oblessing/artisgo/finder"
	lg "github.com/oblessing/artisgo/logger"
	"github.com/oblessing/artisgo/orders"
	"github.com/oblessing/artisgo/platform"
	"github.com/oblessing/artisgo/store/memory"
)

// this params would be injected
var (
	logger    *log2.Logger
	logPrefix = "app:\t"
)

func init() {
	logger = log2.New(os.Stdout, logPrefix, log2.LstdFlags|log2.Lshortfile)
}

func main() {
	ctx := context.Background()

	// Let the system take advantage of all cores.
	runtime.GOMAXPROCS(runtime.NumCPU())

	// Get runtime config
	config, err := settings.GetRuntimeConfig()
	if err != nil {
		logger.Fatal(err)
	}

	// get symbols to trade, retrieve cryptos to monitor
	supportedPairs, err := finder.NewFinderAdapter(config).GetAllUsdtPairs(ctx)
	if err != nil {
		logger.Fatal(err)
	}

	// Create order placing adapter.
	orderAdapter := orders.NewAdapter(config)

	// Create storage, we currently use memory store
	memoryAdapter := memory.NewMemoryStore()

	// Create expert trader
	eaTrader := expert.NewExpertTrader(config, memoryAdapter, orderAdapter)

	lg.Info(ctx, "about to start monitor", zap.Int("count", len(supportedPairs)))

	// start trader
	if err = platform.NewSymbolDatasource(config, eaTrader).StartTrading(ctx, supportedPairs...); err != nil {
		logger.Fatal(err)
	}
}
