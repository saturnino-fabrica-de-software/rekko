package middleware

import (
	"log/slog"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/saturnino-fabrica-de-software/rekko/internal/domain"
)

func TestLastUsedWorker_Enqueue(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	t.Run("enqueues and processes updates", func(t *testing.T) {
		mockRepo := new(MockAPIKeyRepo)
		keyID := uuid.New()

		mockRepo.On("UpdateLastUsed", mock.Anything, keyID).Return(nil)

		config := LastUsedWorkerConfig{
			BufferSize:       10,
			DebounceInterval: 10 * time.Millisecond,
			BatchInterval:    50 * time.Millisecond,
			MaxBatchSize:     5,
		}

		worker := NewLastUsedWorker(mockRepo, logger, config)
		worker.Start()

		worker.Enqueue(keyID)

		time.Sleep(100 * time.Millisecond)

		worker.Stop()

		mockRepo.AssertCalled(t, "UpdateLastUsed", mock.Anything, keyID)
	})

	t.Run("debounces rapid updates for same key", func(t *testing.T) {
		mockRepo := new(MockAPIKeyRepo)
		keyID := uuid.New()

		var callCount int32
		mockRepo.On("UpdateLastUsed", mock.Anything, keyID).Run(func(args mock.Arguments) {
			atomic.AddInt32(&callCount, 1)
		}).Return(nil)

		config := LastUsedWorkerConfig{
			BufferSize:       100,
			DebounceInterval: 1 * time.Second,
			BatchInterval:    50 * time.Millisecond,
			MaxBatchSize:     100,
		}

		worker := NewLastUsedWorker(mockRepo, logger, config)
		worker.Start()

		for i := 0; i < 10; i++ {
			worker.Enqueue(keyID)
		}

		time.Sleep(100 * time.Millisecond)

		worker.Stop()

		assert.Equal(t, int32(1), atomic.LoadInt32(&callCount), "should only update once due to debounce")
	})

	t.Run("processes multiple different keys", func(t *testing.T) {
		mockRepo := new(MockAPIKeyRepo)
		key1 := uuid.New()
		key2 := uuid.New()
		key3 := uuid.New()

		mockRepo.On("UpdateLastUsed", mock.Anything, key1).Return(nil)
		mockRepo.On("UpdateLastUsed", mock.Anything, key2).Return(nil)
		mockRepo.On("UpdateLastUsed", mock.Anything, key3).Return(nil)

		config := LastUsedWorkerConfig{
			BufferSize:       10,
			DebounceInterval: 10 * time.Millisecond,
			BatchInterval:    50 * time.Millisecond,
			MaxBatchSize:     5,
		}

		worker := NewLastUsedWorker(mockRepo, logger, config)
		worker.Start()

		worker.Enqueue(key1)
		worker.Enqueue(key2)
		worker.Enqueue(key3)

		time.Sleep(100 * time.Millisecond)

		worker.Stop()

		mockRepo.AssertCalled(t, "UpdateLastUsed", mock.Anything, key1)
		mockRepo.AssertCalled(t, "UpdateLastUsed", mock.Anything, key2)
		mockRepo.AssertCalled(t, "UpdateLastUsed", mock.Anything, key3)
	})

	t.Run("handles repository errors gracefully", func(t *testing.T) {
		mockRepo := new(MockAPIKeyRepo)
		keyID := uuid.New()

		mockRepo.On("UpdateLastUsed", mock.Anything, keyID).Return(domain.ErrAPIKeyNotFound)

		config := LastUsedWorkerConfig{
			BufferSize:       10,
			DebounceInterval: 10 * time.Millisecond,
			BatchInterval:    50 * time.Millisecond,
			MaxBatchSize:     5,
		}

		worker := NewLastUsedWorker(mockRepo, logger, config)
		worker.Start()

		worker.Enqueue(keyID)

		time.Sleep(100 * time.Millisecond)

		worker.Stop()

		mockRepo.AssertCalled(t, "UpdateLastUsed", mock.Anything, keyID)
	})

	t.Run("drops updates when buffer is full", func(t *testing.T) {
		mockRepo := new(MockAPIKeyRepo)

		mockRepo.On("UpdateLastUsed", mock.Anything, mock.Anything).Return(nil).Maybe()

		config := LastUsedWorkerConfig{
			BufferSize:       2,
			DebounceInterval: 10 * time.Millisecond,
			BatchInterval:    1 * time.Second,
			MaxBatchSize:     100,
		}

		worker := NewLastUsedWorker(mockRepo, logger, config)

		for i := 0; i < 10; i++ {
			worker.Enqueue(uuid.New())
		}

		assert.Equal(t, 2, len(worker.updateCh), "buffer should be capped at size")
	})

	t.Run("processes batch when max size reached", func(t *testing.T) {
		mockRepo := new(MockAPIKeyRepo)
		var callCount int32

		mockRepo.On("UpdateLastUsed", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
			atomic.AddInt32(&callCount, 1)
		}).Return(nil)

		config := LastUsedWorkerConfig{
			BufferSize:       100,
			DebounceInterval: 1 * time.Millisecond,
			BatchInterval:    10 * time.Second,
			MaxBatchSize:     3,
		}

		worker := NewLastUsedWorker(mockRepo, logger, config)
		worker.Start()

		for i := 0; i < 4; i++ {
			worker.Enqueue(uuid.New())
			time.Sleep(5 * time.Millisecond)
		}

		time.Sleep(50 * time.Millisecond)

		worker.Stop()

		assert.GreaterOrEqual(t, atomic.LoadInt32(&callCount), int32(3), "should process when batch is full")
	})
}

func TestDefaultLastUsedWorkerConfig(t *testing.T) {
	config := DefaultLastUsedWorkerConfig()

	assert.Equal(t, 1000, config.BufferSize)
	assert.Equal(t, 1*time.Minute, config.DebounceInterval)
	assert.Equal(t, 5*time.Second, config.BatchInterval)
	assert.Equal(t, 100, config.MaxBatchSize)
}
