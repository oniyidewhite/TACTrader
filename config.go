package config

import (
	"os"
	"strconv"
	"time"
)

var StartTime time.Time

type Config struct {
	BinanceApiKey     string
	BinanceSecretKey  string
	Interval          string
	PercentageLotSize float64
	TradeAmount       float64
	TestType          string
	IsBypass          bool
}

func (c Config) IsTestMode() bool {
	return c.TestType == "test"
}

func GetRuntimeConfig() (Config, error) {
	var data = os.Args[1:]
	if len(data) < 5 {
		// TODO: After testing, we should always return an error.
		return Config{
			Interval:          "5m", // "1m",
			PercentageLotSize: 20,   // 369,
			TradeAmount:       100,
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
	if len(data) == 7 {
		IsBypass = data[6] == "true"
	}

	return Config{
		BinanceApiKey:     data[3],
		BinanceSecretKey:  data[4],
		Interval:          data[0],
		PercentageLotSize: value,
		TradeAmount:       tradeAmount,
		TestType:          data[5],
		IsBypass:          IsBypass,
	}, nil
}

// var (
//	Interval          = "3m"    // "1m" //"5m"  //"1h"//"15m"//"30m"//"15m"
//	PercentageLotSize = 0.14285 //0.011//0.111 //0.811 //0.611//0.369 //1.1 //0.61 // means 1.2% // should be dynamic // 0.14285 or (3.14285), 1.04761
//	TradeAmount       = 100.0   // 10$
// )

// 3m // 0.13258
// 3m 0.14285

// ((22 / 7) / 3) - 1,
// (22/7) - (( (1 + 2 + 3 + 5) / 8))
