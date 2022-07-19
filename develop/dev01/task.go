package main

/*
=== Базовая задача ===

Создать программу печатающую точное время с использованием NTP библиотеки.Инициализировать как go module.
Использовать библиотеку https://github.com/beevik/ntp.
Написать программу печатающую текущее время / точное время с использованием этой библиотеки.

Программа должна быть оформлена с использованием как go module.
Программа должна корректно обрабатывать ошибки библиотеки: распечатывать их в STDERR и возвращать ненулевой код выхода в OS.
Программа должна проходить проверки go vet и golint.
*/

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/beevik/ntp"
)

func main() {
	currentTime, err := getNTPTime()
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	fmt.Println(currentTime)
}

func getNTPTime() (time.Time, error) {
	return ntp.Time("0.beevik-ntp.pool.ntp.org")
}
