package store

import (
	"github.com/globalsign/mgo/bson"
	"time"
)

type BotData struct {
	Candle   *Candle       `json:"candle"`
	IsClosed bool          `json:"is_closed"`
	Date     time.Time     `json:"date"`
	Pair     string        `json:"pair"`
	Id       bson.ObjectId `json:"id" bson:"_id"`
}

type Candle struct {
	Open  float64 `json:"open"`
	Close float64 `json:"close"`
	High  float64 `json:"high"`
	Low   float64 `json:"low"`
	Vol   float64 `json:"vol"`
}

type Database interface {
	// save data
	Save(*BotData) error
	// fetch a specific no of data
	Fetch(string, int) ([]*BotData, error)
}
