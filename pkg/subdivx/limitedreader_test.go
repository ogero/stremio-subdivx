package subdivx

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLimitedReader(t *testing.T) {
	data := []byte("Hello, World!")

	t.Run("Basic Reading", func(t *testing.T) {
		reader := bytes.NewReader(data)
		lr := LimitReader(reader, 5, ErrReadBeyondLimit)
		buf := make([]byte, 5)
		n, err := lr.Read(buf)
		assert.NoError(t, err)
		assert.Equal(t, 5, n)
		assert.Equal(t, "Hello", string(buf))
	})

	t.Run("Read Beyond Limit", func(t *testing.T) {
		reader := bytes.NewReader(data)
		lr := LimitReader(reader, 5, ErrReadBeyondLimit)
		_, err := io.ReadAll(lr)
		assert.ErrorIs(t, err, ErrReadBeyondLimit)
	})

	t.Run("Exact Limit Read", func(t *testing.T) {
		reader := bytes.NewReader(data)
		lr := LimitReader(reader, 14, ErrReadBeyondLimit)
		b, err := io.ReadAll(lr)
		assert.NoError(t, err)
		assert.Equal(t, "Hello, World!", string(b))
	})

	t.Run("EOF Handling", func(t *testing.T) {
		reader := bytes.NewReader(data)
		lr := LimitReader(reader, 20, ErrReadBeyondLimit)
		b, err := io.ReadAll(lr)
		assert.NoError(t, err)
		assert.Equal(t, "Hello, World!", string(b))

	})
}
