package main

/*
=== Утилита telnet ===

Реализовать примитивный telnet клиент:
Примеры вызовов:
go-telnet --timeout=10s host port go-telnet mysite.ru 8080 go-telnet --timeout=3s 1.1.1.1 123

Программа должна подключаться к указанному хосту (ip или доменное имя) и порту по протоколу TCP.
После подключения STDIN программы должен записываться в сокет, а данные полученные и сокета должны выводиться в STDOUT
Опционально в программу можно передать таймаут на подключение к серверу (через аргумент --timeout, по умолчанию 10s).

При нажатии Ctrl+D программа должна закрывать сокет и завершаться. Если сокет закрывается со стороны сервера, программа должна также завершаться.
При подключении к несуществующему сервер, программа должна завершаться через timeout.
*/

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"syscall"
	"time"
)

func main() {
	timeoutFlag := flag.Uint("timeout", 10, "conn timeout")
	flag.Parse()
	timeout := time.Duration(*timeoutFlag) * time.Second
	var host = "localhost"
	var port = "5555"
	args := flag.Args()
	if len(args) > 0 {
		host = args[0]
	}
	if len(args) > 1 {
		port = args[1]
	}
	// устанавливаем tcp-соединение с таймаутом
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%s", host, port), timeout)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	log.Printf("Succesfully connected via %s to %s. Local %s addres is: %s",
		conn.LocalAddr().Network(), conn.RemoteAddr(), conn.LocalAddr().Network(), conn.LocalAddr())
	for {
		fmt.Print(">> ")
		reader := bufio.NewReader(os.Stdin)
		text, err := reader.ReadString('\n')
		if err != nil {
			//закрываем соединение, если было нажато Ctrl+D
			if err == io.EOF {
				log.Println("\r                      \rClosing connection")
				break
			}
			log.Fatal(err)
		}
		// отправляем в соединение
		_, err = fmt.Fprint(conn, text)
		if err != nil {
			// если соединение закрыто
			if errors.Is(err, syscall.EPIPE) {
				log.Println("Connection closed by server")
				break
			}
			log.Fatal(err)
		}
		message, _ := bufio.NewReader(conn).ReadString('\n')
		fmt.Print("->: " + message)
	}
}
