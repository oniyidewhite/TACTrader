package memory

import (
	"context"
	store2 "github.com/oblessing/artisgo/store"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewMemoryStore(t *testing.T) {
	t.Run("should return data in the right order", func(t *testing.T) {
		ctx := context.Background()

		store := NewMemoryStore()
		assert.NoError(t, store.Save(ctx, &store2.BotData{
			Others: map[string]float64{"t": 1},
			Pair:   "Test1",
		}))
		assert.NoError(t, store.Save(ctx, &store2.BotData{
			Others: map[string]float64{"t": 2},
			Pair:   "Test1",
		}))
		assert.NoError(t, store.Save(ctx, &store2.BotData{
			Others: map[string]float64{"t": 3},
			Pair:   "Test1",
		}))
		assert.NoError(t, store.Save(ctx, &store2.BotData{
			Others: map[string]float64{"t": 4},
			Pair:   "Test1",
		}))

		res, err := store.Fetch(ctx, "Test1", 10)
		assert.NoError(t, err)
		assert.Equal(t, float64(1), res[0].Others["t"])

		res, err = store.Fetch(ctx, "Test1", 10)
		assert.NoError(t, err)
		assert.Equal(t, float64(1), res[0].Others["t"])

		res, err = store.Fetch(ctx, "Test1", 2)
		assert.NoError(t, err)
		assert.Equal(t, float64(3), res[0].Others["t"])

		res, err = store.Fetch(ctx, "Test1", 10)
		assert.NoError(t, err)
		assert.Equal(t, float64(2), res[0].Others["t"])
	})
}
