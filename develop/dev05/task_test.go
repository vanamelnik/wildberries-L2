package main

import "testing"

func TestContext(t *testing.T) {
	g := Grep{
		allLines: []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "10"},
		before:   1,
		after:    2,
	}
	goodLines := []int{1, 8}
	goodLines = g.getLinesWithContext(goodLines)
	for _, n := range goodLines {
		t.Log(g.allLines[n])
	}
}
