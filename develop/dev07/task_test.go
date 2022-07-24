package main

import (
	"fmt"
	"math"
	"math/rand"
	"testing"
	"time"
)

// на самом деле этот бенчмарк не репрезентабельный, т.к. or-функция закрывает общий канал при закрытии  любого составляющего её канала.
func BenchmarkOr(b *testing.B) {
	bb := []struct {
		name string
		or   orFn
	}{
		{name: "orReflect", or: orReflect},
		{name: "orRecursive", or: orRecursive},
		{name: "orGoroutines", or: orGoroutines},
	}
	for _, bc := range bb {
		for k := 0.0; k <= 10; k++ {
			n := int(math.Pow(2, k))
			b.Run(fmt.Sprintf("%s/%d", bc.name, n), func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					chans := make([]<-chan interface{}, n)
					for j := 0; j < n; j++ {
						chans[j] = asChan(1, 2, 3, 4, 5, 6, 7, 8, 9, 10)
					}
					for range bc.or(chans...) {
					}
				}
			})
		}
	}
}

func asChan(vs ...int) <-chan interface{} {
	c := make(chan interface{})
	go func() {
		for _, v := range vs {
			c <- v
			time.Sleep(time.Duration(time.Duration(rand.Intn(10)) * time.Millisecond))
		}
		close(c)
	}()
	return c
}
