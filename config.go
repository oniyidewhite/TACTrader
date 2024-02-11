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
	RatioToOne        float64
	BlockSize         int
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
			Interval:          "1m", // "1m",
			PercentageLotSize: 14,   // 369,
			RatioToOne:        0.1,
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
