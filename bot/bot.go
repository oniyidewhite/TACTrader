package bot

import "time"

// logic to connect to binance
// should be generic so we can connect to any platform
// some systems uses spreads

// const

// var

// structs
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
	IsTest bool
}

type TradeBot interface {
	// observe the source for any
	// called by the system
	OnCreate(*Config)
}

//// Called once we have new candle stick
//// Called by the owner
//OnResult(*Candle) error
//// Called when our previously opened trade is closed
//// Called by the owner
//OnClose(symbol string) error
//// Make trade
//// called by the system
//MakeTrade(*Args) error

// inits


// functions
