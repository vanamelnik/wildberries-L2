package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_getNTPTime(t *testing.T) {
	got, err := getNTPTime()
	local := time.Now()
	assert.NoError(t, err)
	delta := got.Sub(local)
	delta = abs(delta)
	t.Logf("the difference between NTP and local time is %d = %v\n", delta, delta)
	assert.Less(t, delta, time.Millisecond*500)
}

func abs(d time.Duration) time.Duration {
	n := int(d)
	if n < 0 {
		n = -n
	}
	return time.Duration(n)
}
