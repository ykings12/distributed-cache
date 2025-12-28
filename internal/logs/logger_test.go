package logs

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLogger(t *testing.T) {
	t.Run("LevelFiltering", func(t *testing.T) {
		logger := NewLogger(10, INFO)
		// Minimum level is INFO
		logger.Debug("should not be logged")
		logger.Info("should be logged")
		logger.Warn("should be logged")
		logger.Error("should be logged")

		entries := logger.GetLast(10)
		assert.Len(t, entries, 3, "Logger should have ignored DEBUG but kept INFO, WARN, and ERROR")
		assert.Equal(t, INFO, entries[0].Level)
		assert.Equal(t, WARN, entries[1].Level)
		assert.Equal(t, ERROR, entries[2].Level)
	})

	t.Run("RingBufferBehavior", func(t *testing.T) {
		// max size is 2 so adding a 3rd entry shall push out the first entry (FIFO)
		logger := NewLogger(2, DEBUG)

		logger.Info("first")
		logger.Info("second")
		logger.Info("third")

		entries := logger.GetLast(10)
		assert.Len(t, entries, 2, "Logger should only keep maxSize entries")
		assert.Equal(t, "second", entries[0].Message)
		assert.Equal(t, "third", entries[1].Message)

	})

	t.Run("ConcurrentLogging", func(t *testing.T) {
		//50 different goroutines logging simultaneously
		logger := NewLogger(100, DEBUG)
		var wg sync.WaitGroup
		numLogs := 50

		for i := 0; i < numLogs; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				logger.Info("concurrent log " + string(rune(i)))
			}(i)
		}
		wg.Wait()

		entries := logger.GetLast(100)
		assert.Len(t, entries, numLogs, "Logger should have all concurrent log entries")
	})

	t.Run("GetLastBoundaries,", func(t *testing.T) {
		//3 logs in memory
		//test requesting more, equal and less than available logs
		logger := NewLogger(10, DEBUG)
		logger.Info("msg1")
		logger.Info("msg2")
		logger.Info("msg3")

		//case 1: request more than available (should return all 3)
		assert.Len(t, logger.GetLast(10), 3)

		//case 2: request exactly available (should return all 3)
		assert.Len(t, logger.GetLast(3), 3)

		//case 3: request less than available (should return last 2)
		lastTwo := logger.GetLast(2)
		assert.Len(t, lastTwo, 2)
		assert.Equal(t, "msg2", lastTwo[0].Message)
		assert.Equal(t, "msg3", lastTwo[1].Message)

	})

	t.Run("DeepCopyProtection", func(t *testing.T) {
		logger := NewLogger(10, DEBUG)
		logger.Info("original message")

		entries := logger.GetLast(1)
		entries[0].Message = "modified message"

		entriesAfterModification := logger.GetLast(1)
		assert.Equal(t, "original message", entriesAfterModification[0].Message, "Modifying retrieved entries should not affect internal log storage")
	})
}
