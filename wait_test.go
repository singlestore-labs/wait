package wait_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/singlestore-labs/wait"
)

type logger struct {
	t   *testing.T
	log []log
}

type log struct {
	line string
	fmt  string
	args []any
}

func (l *logger) Log(format string, args ...any) {
	l.log = append(l.log, log{
		line: fmt.Sprintf(format, args...),
		fmt:  format,
		args: args,
	})
}

func TestWaitQuick(t *testing.T) {
	t.Parallel()
	logger := &logger{t: t}

	t.Log("no wait required")
	start := time.Now()
	var count int
	require.NoError(t, wait.For(func() (bool, error) {
		count++
		return true, nil
	}, wait.WithLogger(logger.Log)))
	assert.Equal(t, 1, count)
	assert.Less(t, time.Since(start), time.Millisecond*200*windowsMult())

	// done := make(chan struct{})

	t.Log("some wait required")
	start = time.Now()
	count = 0
	require.NoError(t, wait.For(func() (bool, error) {
		count++
		return count >= 3, nil
	}, wait.WithLogger(logger.Log), wait.WithInterval(time.Microsecond*windowsMult())))
	assert.Equal(t, 3, count)
	assert.Less(t, time.Since(start), time.Millisecond*200*windowsMult())
}

func TestWaitIncrease(t *testing.T) {
	t.Parallel()
	logger := &logger{t: t}
	start := time.Now()
	count := 0
	prior := time.Now()
	intervals := make([]time.Duration, 10)
	require.NoError(t, wait.For(func() (bool, error) {
		this := time.Now()
		intervals[count] = this.Sub(prior)
		prior = this
		count++
		return count >= 10, nil
	}, wait.WithLogger(logger.Log), wait.WithInterval(time.Millisecond*5*windowsMult()), wait.WithBackoff(1.4)))
	assert.Equal(t, 10, count)
	assert.Less(t, time.Since(start), time.Millisecond*200*windowsMult())
	firstThree := intervals[0] + intervals[1] + intervals[2]
	lastThree := intervals[7] + intervals[8] + intervals[9]
	t.Logf(" sum of first three intervals: %s, last three intervals: %s", firstThree, lastThree)
	assert.Less(t, firstThree, lastThree)
}

func isWindows() bool {
	return os.PathSeparator == '\\' && os.PathListSeparator == ';'
}

func windowsMult() time.Duration {
	if isWindows() {
		return 10
	}
	return 1
}
