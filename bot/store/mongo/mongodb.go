package mongo

import (
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/oblessing/artisgo/bot/store"
	"log"
)

const (
	db         = "tradingBot"
	collection = "candlesticks"
)

type MongoDBService struct {
	collection *mgo.Collection
}

// save data
func (m MongoDBService) Save(data *store.BotData) error {
	return m.collection.Insert(data)
}

// fetch a specific no of data
func (m MongoDBService) Fetch(pair string, size int) ([]*store.BotData, error) {
	data := []*store.BotData{}
	err := m.collection.Find(bson.M{"pair": pair}).Sort("-date").Limit(size).All(&data)

	return data, err
}

func startMongoService(url string) (*mgo.Collection, error) {
	session, err := mgo.Dial(url)
	if err != nil {
		return nil, err
	}
	return session.DB(db).C(collection), nil
}

func NewMongoInstance(url string, logger *log.Logger) store.Database {
	collection, err := startMongoService(url)
	if err != nil {
		logger.Fatal(err)
	}
	var mongodb interface{} = MongoDBService{collection}
	return mongodb.(store.Database)
}
