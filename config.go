package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/kelseyhightower/envconfig"
)

var StartTime time.Time

type Config struct {
	BinanceApiKey     string  `envconfig:"BINANCE_API_KEY"`
	BinanceSecretKey  string  `envconfig:"BINANCE_SECRET_KEY"`
	Interval          string  `envconfig:"INTERVAL" default:"3m"`
	PercentageLotSize float64 `envconfig:"PERCENTAGE_LOT_SIZE" default:"14"`
	RatioToOne        float64 `envconfig:"RATIO_TO_ONE" default:"0.07"`
	BlockSize         int     `envconfig:"BLOCK_SIZE" default:"10"`
	TradeAmount       float64 `envconfig:"TRADE_AMOUNT" default:"40"`
	TestType          string  `envconfig:"TEST_TYPE" default:"real"`
	IsBypass          bool    `envconfig:"IS_BYPASS" default:"false"`
}

func (c Config) IsTestMode() bool {
	return c.TestType == "test"
}

// GetRuntimeConfig returns the config to be used on app start
// DEPRECATED: we might need this again since we're deploying to cloud
func GetRuntimeConfig() (Config, error) {
	var data = os.Args[1:]
	if len(data) < 5 {
		// TODO: After testing, we should always return an error.
		return Config{
			BinanceApiKey:     "",
			BinanceSecretKey:  "",
			Interval:          "3m", // "1m",
			PercentageLotSize: 14,   // 369,
			RatioToOne:        0.07,
			BlockSize:         10,
			TradeAmount:       40,
			TestType:          "real",
			IsBypass:          false,
		}, nil
	}

	value, err := strconv.ParseFloat(data[1], 64)
	if err != nil {
		return Config{}, err
	}

	tradeAmount, err := strconv.ParseFloat(data[2], 64)
	if err != nil {
		return Config{}, err
	}

	IsBypass := false
	if len(data) >= 7 {
		IsBypass = data[6] == "true"
	}

	RatioToOne := 1.5207418
	if len(data) >= 8 {
		RatioToOne, _ = strconv.ParseFloat(data[7], 64)
	}

	BlockSize := 5
	if len(data) >= 9 {
		v, _ := strconv.ParseFloat(data[8], 64)

		BlockSize = int(v)
	}

	return Config{
		BinanceApiKey:     data[3],
		BinanceSecretKey:  data[4],
		Interval:          data[0],
		PercentageLotSize: value,
		RatioToOne:        RatioToOne,
		BlockSize:         BlockSize,
		TradeAmount:       tradeAmount,
		TestType:          data[5],
		IsBypass:          IsBypass,
	}, nil
}

// Load loads up the config into the app start initialization
func Load() (Config, error) {
	var cfg Config

	if err := envconfig.Process("", &cfg); err != nil {
		return Config{}, fmt.Errorf("error loading config: %w", err)
	}

	return cfg, nil
}
