package bot

import "time"

type Candle struct {
	Pair  float64
	Open  float64
	Close float64
	High  float64
	Low   float64
	Vol   float64
	Time  time.Time
}

type Args struct {
	Time       time.Time
	Pair       string
	Open       float64
	StopLoss   float64
	TakeProfit float64
}

// If Period changes for a Pair, do clear existing records from db
type Config struct {
	Pair, Period string
	IsTest       bool
}

type TradeBot interface {
	// observe the source for any
	// called by the system
	OnCreate(*Config)
}
