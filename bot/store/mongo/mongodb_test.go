package mongo

import (
	"github.com/globalsign/mgo/bson"
	"github.com/oblessing/artisgo/bot/store"
	"log"
	"math/rand"
	"os"
	"testing"
)

const mongoUri = "mongodb://admin:password@127.0.0.1:27017/admin"

var (
	config = &Config{
		url:        mongoUri,
		db:         "tradingTest",
		collection: "candlesticks",
	}
	logger = log.New(os.Stdout, "test:\t", log.LstdFlags|log.Lshortfile)
)

func TestConnection(t *testing.T) {
	_, err := NewMongoInstance("test")
	if err == nil {
		t.Fatalf("mongo service should not start when address is invalid")
	}
}

func TestMongoDBService_Fetch(t *testing.T) {
	service, err := NewMongoInstanceWithLogger(logger, config)
	if err != nil {
		t.Fatalf("Unable to initialize mongo service: %+v", err)
	}
	records, err := service.Fetch("test", 1)
	if err != nil {
		t.Fatalf("Unable to retrieve records: %+v", err)
	}

	if records == nil {
		t.Fatalf("Records can not be nil")
	}
}

func TestMongoDBService_Save(t *testing.T) {
	service, err := NewMongoInstanceWithLogger(logger, config)
	if err != nil {
		t.Fatalf("Unable to initialize mongo service: %+v", err)
	}

	err = service.Save(&store.BotData{
		Candle: randomCandle(),
		Date:   bson.Now(),
		Pair:   "test",
		Id:     bson.NewObjectId(),
	})

	if err != nil {
		t.Fatalf("Unable to save record to db:%+v", err)
	}
}

func TestMongoDBService_SaveAndRetrieve(t *testing.T) {
	limit := 2
	pair := "test"

	service, err := NewMongoInstanceWithLogger(logger, config)
	if err != nil {
		t.Fatalf("Unable to initialize mongo service: %+v", err)
	}

	ids := []bson.ObjectId{bson.NewObjectId(), bson.NewObjectId()}

	err = service.Save(&store.BotData{
		Candle: randomCandle(),
		Date:   bson.Now(),
		Pair:   pair,
		Id:     bson.NewObjectId(),
	})

	err = service.Save(&store.BotData{
		Candle: randomCandle(),
		Date:   bson.Now(),
		Pair:   pair,
		Id:     bson.NewObjectId(),
	})

	err = service.Save(&store.BotData{
		Candle: randomCandle(),
		Date:   bson.Now(),
		Pair:   pair,
		Id:     bson.NewObjectId(),
	})

	err = service.Save(&store.BotData{
		Candle: randomCandle(),
		Date:   bson.Now(),
		Pair:   pair,
		Id:     ids[0],
	})

	err = service.Save(&store.BotData{
		Candle: randomCandle(),
		Date:   bson.Now(),
		Pair:   pair,
		Id:     ids[1],
	})

	if err != nil {
		t.Fatalf("Unable to save record to db:%+v", err)
	}

	records, err := service.Fetch(pair, limit)
	if err != nil {
		t.Fatalf("Unable to retrieve records: %+v", err)
	}

	if len(records) != limit {
		t.Fatalf("Only 2 records should be returned: %+v", err)
	}

	data := []bson.ObjectId{}
	for _, v := range records {
		data = append(data, v.Id)
	}

	if data[0] != ids[1] {
		t.Fatalf("Invalid order, got:%+v expected %+v", data, ids)
	}

	if data[1] != ids[0] {
		t.Fatalf("Invalid order, got:%+v expected %+v", data, ids)
	}
}

func randomCandle() *store.Candle {
	return &store.Candle{
		Open:  rand.Float64(),
		Close: rand.Float64(),
		High:  rand.Float64(),
		Low:   rand.Float64(),
		Vol:   rand.Float64(),
	}
}
