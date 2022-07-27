package main

// telnet эхо-сервер для проверки клиента

import (
	"syscall"
	"time"

	"github.com/reiver/go-telnet"
)

func main() {
	handler := telnet.EchoHandler
	time.AfterFunc(3*time.Second, func() { syscall.Kill(syscall.Getpid(), syscall.SIGINT) })
	err := telnet.ListenAndServe(":5555", handler)
	if nil != err {
		panic(err)
	}
}
