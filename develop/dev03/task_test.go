package main

import (
	"math/rand"
	"testing"
	"time"

	"github.com/bxcodec/faker/v3"
	"github.com/stretchr/testify/assert"
)

func TestSort(t *testing.T) {
	faker.SetRandomSource(rand.NewSource(time.Now().UnixNano()))
	const numLines = 100
	text := randomLines(numLines)
	t.Run("Normal test", func(t *testing.T) {
		got, err := (&GoSorter{}).Sort(text)
		assert.NoError(t, err)

	})
}

func randomLines(n int) []string {
	result := make([]string, 0, n)
	for i := 0; i < n; i++ {
		result = append(result, faker.Sentence())
	}
	return result
}
