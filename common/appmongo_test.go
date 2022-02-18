package common

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

const mongoUri = "mongodb://user:password@localhost:27017"

func TestNewDriver(t *testing.T) {
	t.Run("should return error", func(t *testing.T) {
		_, err := NewDriver("test")
		assert.Error(t, err)
	})

	t.Run("should connect", func(t *testing.T) {
		rr, err := NewDriver(mongoUri)
		assert.NoError(t, err)
		assert.NotNil(t, rr)
	})
}
