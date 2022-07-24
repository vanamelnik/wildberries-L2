package main

import (
	"fmt"
	"log"
	"reflect"
	"sync"
	"time"
)

/*
=== Or channel ===

Реализовать функцию, которая будет объединять один или более done каналов в single канал если один из его составляющих каналов закроется.
Одним из вариантов было бы очевидно написать выражение при помощи select, которое бы реализовывало эту связь,
однако иногда неизестно общее число done каналов, с которыми вы работаете в рантайме.
В этом случае удобнее использовать вызов единственной функции, которая, приняв на вход один или более or каналов, реализовывала весь функционал.

Определение функции:
var or func(channels ...<- chan interface{}) <- chan interface{}

Пример использования функции:
sig := func(after time.Duration) <- chan interface{} {
	c := make(chan interface{})
	go func() {
		defer close(c)
		time.Sleep(after)
}()
return c
}

start := time.Now()
<-or (
	sig(2*time.Hour),
	sig(5*time.Minute),
	sig(1*time.Second),
	sig(1*time.Hour),
	sig(1*time.Minute),
)

fmt.Printf(“fone after %v”, time.Since(start))
*/

// orFn - функция, объединяющая произвольное количество каналов в один канал, который закрывается при закрытии любого из предоставленных каналов.
type orFn func(channels ...<-chan interface{}) <-chan interface{}

// orGourutines - реализация описанной функции, в которой на каждый or-channel выделяется своя горутина.
func orGoroutines(channels ...<-chan interface{}) <-chan interface{} {
	outCh := make(chan interface{})
	wg := &sync.WaitGroup{}
	wg.Add(len(channels))
	mu := &sync.Mutex{}
	for _, ch := range channels {
		go func(ch <-chan interface{}) {
			for v := range ch {
				mu.Lock()
				outCh <- v
				mu.Unlock()
			}
			mu.Lock()
			if outCh != nil {
				close(outCh)
				outCh = nil
			}
			mu.Unlock()
			wg.Done()
		}(ch)
	}
	go func() {
		wg.Wait()
		if outCh != nil {
			close(outCh)
			outCh = nil
		}
	}()
	return outCh
}

// orRecursive - or-функция, рекурсивно мержащая по два канала. Должна работать быстрее, чем orGoroutines,
// т.к. количесвто горутин ~ log2 от количества каналов => меньше накладных расходов на переключение горутин.
// Но при большом количестве каналов возникают бОльшие накладные расходы на память в стеке, связанные с рекурсией.
func orRecursive(channels ...<-chan interface{}) <-chan interface{} {
	switch len(channels) {
	case 0:
		ch := make(chan interface{})
		close(ch)
		return ch
	case 1:
		return channels[0]
	default:
		n := len(channels) / 2
		return orTwo(
			orRecursive(channels[:n]...),
			orRecursive(channels[n:]...))
	}
}

// orTwo - вспомогательная функция, мержащая два канала.
func orTwo(a, b <-chan interface{}) <-chan interface{} {
	outCh := make(chan interface{})
	go func() {
	loop:
		for {
			select {
			case v, ok := <-a:
				if !ok {
					break loop
				}
				outCh <- v
			case v, ok := <-b:
				if !ok {
					break loop
				}
				outCh <- v
			}
		}
		close(outCh)
	}()
	return outCh
}

// orReflect - or-функция использующая reflect.Select. Как и всё, связанное с рефлексией, должна работать медленнее аналогов.
func orReflect(channels ...<-chan interface{}) <-chan interface{} {
	outCh := make(chan interface{})
	var cases []reflect.SelectCase
	for _, ch := range channels {
		cases = append(cases, reflect.SelectCase{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(ch),
		})
	}
	go func() {
		for {
			_, v, ok := reflect.Select(cases)
			if !ok {
				break
			}
			outCh <- v.Interface()
		}
		close(outCh)
	}()
	return outCh
}

// тестируем работоспособность всех вариантов
func main() {
	log.Println("Testing orReflect...")
	testOr1(orReflect)
	testOr2(orReflect)
	log.Println("Testing orGoroutines...")
	testOr1(orGoroutines)
	testOr2(orGoroutines)
	log.Println("Testing orRecursive...")
	testOr1(orRecursive)
	testOr2(orRecursive)
}

// testOr1 - тест из задания.
func testOr1(or orFn) {
	log.Println("Test1")
	sig := func(after time.Duration) <-chan interface{} {
		c := make(chan interface{})
		go func() {
			defer close(c)
			time.Sleep(after)
		}()
		return c
	}

	start := time.Now()
	<-or(
		sig(2*time.Hour),
		sig(5*time.Minute),
		sig(1*time.Second),
		sig(1*time.Hour),
		sig(1*time.Minute),
	)

	fmt.Printf("done after %v\n", time.Since(start))
}

// testOr2 показывает, что пока один из каналов не закрыт, данные из всех каналов объединяются.
func testOr2(or orFn) {
	log.Println("Test2")
	tickFn := func(msg string, d time.Duration) <-chan interface{} {
		ch := make(chan interface{})
		t := time.NewTicker(d)
		go func() {
			for range t.C {
				ch <- msg
			}
		}()
		return ch
	}
	stopFn := func(d time.Duration) <-chan interface{} {
		ch := make(chan interface{})
		go func() {
			ch <- fmt.Sprintf("stopFn: через %v закрою мой канал!\n", d)
			time.Sleep(d)
			ch <- "\nstopFn: закрываю канал!\n"
			close(ch)
		}()
		return ch
	}
	for msg := range or(
		tickFn("тик ", time.Second/2),
		tickFn("так ", time.Second/4),
		tickFn("БУМ!!!\n", time.Second),
		stopFn(time.Second*5),
		stopFn(time.Second*10),
	) {
		fmt.Print(msg)
	}
	fmt.Println("Ждем непонятно чего...")
	time.Sleep(time.Second * 3)
}
