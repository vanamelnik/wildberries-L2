package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"

	gops "github.com/mitchellh/go-ps"
)

/*
=== Взаимодействие с ОС ===

Необходимо реализовать собственный шелл

встроенные команды: cd/pwd/echo/kill/ps
поддержать fork/exec команды
конвеер на пайпах

Реализовать утилиту netcat (nc) клиент
принимать данные из stdin и отправлять в соединение (tcp/udp)
Программа должна проходить все тесты. Код должен проходить проверки go vet и golint.
*/

var (
	// ErrExit возвращается командой exit.
	ErrExit = errors.New("exit")
)

// ErrIncorrectCommand возвращается при невозможности выполнить команду.
func ErrIncorrectCommand(cmd string) error {
	return fmt.Errorf("Bad command or file name: %s", cmd)
}

// функция команды
type cmdFunc func(string) error

// диспетчер команд
var cmdMap = map[string]cmdFunc{
	"exit": exit,
	"pwd":  pwd,
	"cd":   os.Chdir,
	"echo": echo,
	"ps":   ps,
	"kill": kill,
}

// cmdLine представляет командную строку, введенную пользователем.
type cmdLine struct {
	// команда
	cmd string
	// аргументы
	args string
	// следующая команда в пайпе
	nextCmd *cmdLine
}

// Exec запускает команду.
func (cl cmdLine) Exec(stdin *os.File) error {
	command, ok := cmdMap[cl.cmd]
	if !ok {
		return ErrIncorrectCommand(cl.cmd)
	}
	return command(cl.args)
}

// parseCmdLine переводит строку, введенную пользователем в структуру команды
func parseCmdLine(c string) *cmdLine {
	c = strings.TrimSpace(c)
	pipe := strings.Split(c, "|")
	var nextCmd *cmdLine = nil
	if len(pipe) > 1 {
		nextCmd = parseCmdLine(strings.Join(pipe[1:], "|"))
	}
	fields := strings.Fields(c)
	cl := cmdLine{
		cmd:     fields[0],
		nextCmd: nextCmd,
	}
	if len(fields) > 1 {
		cl.args = strings.Join(fields[1:], " ")
	}
	return &cl
}

// printPrompt выводит приглашение командной строки.
func printPrompt() {
	dir, _ := os.Getwd()
	fmt.Printf("%s$ ", dir)
}

// выводит строку на экран
func echo(s string) error {
	fmt.Println(s)
	return nil
}

// ps выводит список процессов
func ps(s string) error {
	processes, err := gops.Processes()
	if err != nil {
		return err
	}
	fmt.Println("PID\t\tname")
	fmt.Println("------------------------")
	for _, p := range processes {
		fmt.Printf("%d\t\t%s\n", p.Pid(), p.Executable())
	}
	return nil
}

// pwd выводит текущий путь
func pwd(s string) error {
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	fmt.Println(dir)
	return nil
}

// kill убивает процесс с данным pid.
func kill(pidStr string) error {
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return err
	}
	return syscall.Kill(pid, syscall.SIGINT)
}

// exit завершает работу оболочки.
func exit(string) error {
	return ErrExit
}

// parser обрабатывает строку, введенную пользователем.
func parser(c string) error {
	if c == "" || c == "\n" {
		return nil
	}
	cl := parseCmdLine(c)
	return cl.Exec(os.Stdin)
}

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("Welcome to gosh!")
	printPrompt()
	for scanner.Scan() {
		if err := parser(scanner.Text()); err != nil {
			if errors.Is(err, ErrExit) {
				break
			}
			fmt.Fprintln(os.Stderr, err)
		}
		printPrompt()
	}
	fmt.Println("Bye!")
}
