package memory

import (
	"context"
	"fmt"
	"sync"

	"github.com/oblessing/artisgo/store"
)

const maxSize = 10

type tmpStorageData struct {
	data *store.BotData
	prev *tmpStorageData
}

type tmpStorage struct {
	lock  sync.RWMutex
	store map[string]tmpStorageData
}

func NewMemoryStore() store.Database {
	db := tmpStorage{}
	db.cleanup()

	return &db
}

func (m *tmpStorage) get(pair string) (tmpStorageData, error) {
	var result tmpStorageData
	var err error
	m.lock.Lock()
	data, ok := m.store[pair]
	if !ok {
		err = fmt.Errorf("no record for: %s", pair)
	}
	result = data
	m.lock.Unlock()

	return result, err
}

func (m *tmpStorage) save(pair string, record tmpStorageData) error {
	m.lock.Lock()
	m.store[pair] = record
	m.lock.Unlock()

	return nil
}

// Fetch returns the requested size, note, fetch is self resizing since we don't have a way yet to keep overall data size in check
func (m *tmpStorage) Fetch(ctx context.Context, pair string, size int) ([]*store.BotData, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	var d tmpStorageData
	var err error
	d, err = m.get(pair)
	if err != nil {
		return nil, err
	}

	var count = 0
	var pointer = &d
	result := []*store.BotData{}
	for {
		// Check if we have reached the size we want.
		// if so and pointer != nil, set prev to nil
		if count == size && pointer != nil {
			pointer.prev = nil
			break
		}

		// Check if we have no more data
		if pointer == nil {
			break
		}

		result = append(result, pointer.data)
		pointer = pointer.prev
		count += 1
	}

	final := []*store.BotData{}
	for i := len(result) - 1; i >= 0; i = i - 1 {
		final = append(final, result[i])
	}

	return final, nil
}

func (m *tmpStorage) Save(ctx context.Context, candle *store.BotData) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	var newData tmpStorageData
	record, err := m.get(candle.Pair)
	if err == nil {
		newData.prev = &record
	}
	newData.data = candle

	return m.save(candle.Pair, newData)
}

func (m *tmpStorage) cleanup() {
	m.store = map[string]tmpStorageData{}
}

// Save(context.Context, *BotData) error
// // Fetch retrieves record from database
// Fetch(context.Context, string, int) ([]*BotData, error)
