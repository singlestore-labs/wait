package wait_test

// This file generated using Claude 3.7 Sonnet

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/singlestore-labs/wait"
)

func TestWaitTimeout(t *testing.T) {
	t.Parallel()
	logger := &logger{t: t}

	t.Log("Testing timeout behavior")
	start := time.Now()
	var count int
	err := wait.For(func() (bool, error) {
		count++
		return false, nil // Always returns false to force timeout
	}, wait.WithLogger(logger.Log), wait.WithLimit(time.Millisecond*100*windowsMult()), wait.WithInterval(time.Millisecond*10*windowsMult()))

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "gave up after")
	assert.GreaterOrEqual(t, count, 2) // Should have tried at least a few times
	elapsed := time.Since(start)
	assert.GreaterOrEqual(t, elapsed, time.Millisecond*100*windowsMult())
	assert.Less(t, elapsed, time.Millisecond*300*windowsMult()) // Allow some leeway
}

func TestWaitContextCancellation(t *testing.T) {
	t.Parallel()
	logger := &logger{t: t}

	t.Log("Testing context cancellation")
	ctx, cancel := context.WithCancel(context.Background())
	
	var count int32
	errCh := make(chan error)
	
	go func() {
		errCh <- wait.For(func() (bool, error) {
			atomic.AddInt32(&count, 1)
			time.Sleep(time.Millisecond * 5 * windowsMult())
			return false, nil // Always return false to keep waiting
		}, wait.WithLogger(logger.Log), 
		   wait.WithContext(ctx),
		   wait.WithLimit(time.Second*30),
		   wait.WithInterval(time.Millisecond*10*windowsMult()))
	}()
	
	// Give it some time to run a few iterations
	time.Sleep(time.Millisecond * 50 * windowsMult())
	cancel() // Cancel the context
	
	err := <-errCh
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")
	assert.Greater(t, atomic.LoadInt32(&count), int32(1)) // Should have run at least once
}

func TestWaitErrorHandling(t *testing.T) {
	t.Parallel()
	logger := &logger{t: t}

	t.Log("Testing error handling - default behavior")
	expectedErr := errors.New("test error")
	count := 0
	err := wait.For(func() (bool, error) {
		count++
		return false, expectedErr // Return error but not success
	}, wait.WithLogger(logger.Log), wait.WithLimit(time.Millisecond*100*windowsMult()), wait.WithInterval(time.Millisecond*10*windowsMult()))

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "gave up after")
	assert.Contains(t, err.Error(), expectedErr.Error())
	assert.Greater(t, count, 1) // Should continue despite errors with default settings

	t.Log("Testing error handling - exit on error")
	count = 0
	err = wait.For(func() (bool, error) {
		count++
		return false, expectedErr
	}, wait.WithLogger(logger.Log), wait.ExitOnError(true), wait.WithInterval(time.Millisecond*10*windowsMult()))

	assert.Error(t, err)
	assert.Equal(t, expectedErr, err) // Should return the actual error
	assert.Equal(t, 1, count) // Should exit immediately on error
}

func TestWaitMaxInterval(t *testing.T) {
	t.Parallel()
	logger := &logger{t: t}

	t.Log("Testing max interval behavior")
	var timestamps []time.Time
	
	err := wait.For(func() (bool, error) {
		now := time.Now()
		timestamps = append(timestamps, now)
		return len(timestamps) >= 5, nil
	}, wait.WithLogger(logger.Log),
	   wait.WithMinInterval(time.Millisecond*10*windowsMult()),
	   wait.WithMaxInterval(time.Millisecond*30*windowsMult()),
	   wait.WithBackoff(2.0)) // Aggressive backoff to hit max quickly

	require.NoError(t, err)
	require.GreaterOrEqual(t, len(timestamps), 5)
	
	// Check that later intervals don't exceed max
	for i := 2; i < len(timestamps)-1; i++ {
		interval := timestamps[i+1].Sub(timestamps[i])
		assert.LessOrEqual(t, interval, time.Millisecond*40*windowsMult()) // Max plus some margin
	}
}

func TestWaitReporting(t *testing.T) {
	t.Parallel()
	logger := &logger{t: t}

	t.Log("Testing reporting functionality")
	reportCount := 0
	customReporter := func(opts wait.O, startTime time.Time) {
		reportCount++
		logger.Log("Report #%d after %s", reportCount, time.Since(startTime))
	}
	
	err := wait.For(func() (bool, error) {
		return reportCount >= 3, nil // Stop after 3 reports
	}, wait.WithLogger(logger.Log),
	   wait.WithReporter(customReporter),
	   wait.WithReports(3), // Request approximately 3 reports
	   wait.WithLimit(time.Second*3),
	   wait.WithInterval(time.Millisecond*20*windowsMult()))
	
	require.NoError(t, err)
	assert.Equal(t, 3, reportCount)
	assert.GreaterOrEqual(t, len(logger.log), 3) // Should have at least 3 log entries
}

func TestWaitDescription(t *testing.T) {
	t.Parallel()
	logger := &logger{t: t}

	t.Log("Testing custom description")
	customDesc := "custom test condition"
	
	err := wait.For(func() (bool, error) {
		return false, nil
	}, wait.WithLogger(logger.Log),
	   wait.WithDescription(customDesc),
	   wait.WithLimit(time.Millisecond*50*windowsMult()),
	   wait.WithInterval(time.Millisecond*10*windowsMult()))
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), customDesc)
	
	// Check that it shows up in logs too
	var foundInLogs bool
	for _, entry := range logger.log {
		if entry.fmt == "%s-%s wait for %s, in progress" {
			assert.Equal(t, customDesc, entry.args[2])
			foundInLogs = true
			break
		}
	}
	assert.True(t, foundInLogs, "Custom description should appear in logs")
}

func TestWaitSuccessWithError(t *testing.T) {
	t.Parallel()
	logger := &logger{t: t}

	t.Log("Testing success with error")
	expectedErr := errors.New("expected error")
	
	err := wait.For(func() (bool, error) {
		return true, expectedErr // Success but with error
	}, wait.WithLogger(logger.Log))
	
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
}

func TestWaitCombinedOptions(t *testing.T) {
	t.Parallel()
	logger := &logger{t: t}

	t.Log("Testing combined options")
	start := time.Now()
	var count int
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	err := wait.For(func() (bool, error) {
		count++
		return count >= 5, nil
	}, wait.WithLogger(logger.Log),
	   wait.WithContext(ctx),
	   wait.WithLimit(time.Second),
	   wait.WithInterval(time.Millisecond*5*windowsMult()),
	   wait.WithBackoff(1.05),
	   wait.WithDescription("combined test"),
	   wait.WithReports(2))
	
	require.NoError(t, err)
	assert.Equal(t, 5, count)
	assert.Less(t, time.Since(start), time.Millisecond*200*windowsMult())
}
