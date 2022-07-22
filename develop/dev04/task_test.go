package main

import (
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAnagrams(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	// генерируем случайное число групп анаграмм по 5, по 2 в каждой группе и единичные слова.
	num5 := rand.Intn(5) + 1
	num2 := rand.Intn(5) + 1
	num1 := rand.Intn(10) + 1
	arr := make([]string, 0, num5+num2+num1)
	arr = append(arr, generateAnagrams(5, num5)...)
	arr = append(arr, generateAnagrams(2, num2)...)
	arr = append(arr, generateAnagrams(1, num1)...)
	shuffle(arr)
	m := GetAnagrams(&arr)
	var got5, got2 int
	// считаем анаграммы по группам
	for _, v := range m {
		if len(v) == 4 {
			got5++
			continue
		}
		if len(v) == 1 {
			got2++
			continue
		}
		t.Errorf("incorrect output: %v", v)
	}
	assert.Equal(t, num5, got5)
	assert.Equal(t, num2, got2)
}

func generateAnagrams(anagramsInGroup, numOfGroups int) []string {
	randLetter := func() string {
		var letters = []rune("йцукенгшщзхъфывапролджэячсмитьбю")
		i := rand.Intn(len(letters))
		return string(letters[i])
	}
	result := make([]string, 0, anagramsInGroup*numOfGroups)
	for i := 0; i < numOfGroups; i++ {
		wordLength := rand.Intn(5) + 4
		word := ""
		for j := 0; j < wordLength; j++ {
			word += randLetter()
		}
		w := []rune(word)
		group := make(map[string]struct{})
		for len(group) < anagramsInGroup {
			shuffle(w)
			group[string(w)] = struct{}{}
		}
		for x := range group {
			result = append(result, x)
		}
	}
	return result
}

func shuffle[T comparable](arr []T) {
	for q := 0; q < rand.Intn(100); q++ {
		i := rand.Intn(len(arr))
		j := rand.Intn(len(arr))
		arr[i], arr[j] = arr[j], arr[i]
	}
}
