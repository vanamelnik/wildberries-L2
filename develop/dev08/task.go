package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
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

type (
	// контекст выполнения команды
	cmdContext struct {
		stdin  io.Reader
		stdOut io.Writer
		args   string
	}

	// функция команды
	cmdFunc func(cmdContext) error
)

// диспетчер команд
var cmdMap = map[string]cmdFunc{
	"exit": exit,
	"pwd":  pwd,
	"cd":   cd,
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

// Execute запускает команду.
func (cl cmdLine) Execute(stdIn io.Reader, stdOut io.Writer) error {
	buf := bytes.Buffer{}
	ctx := cmdContext{
		stdin:  stdIn,
		stdOut: stdOut,
		args:   cl.args,
	}
	// если в пайпе есть команда дальше - буферизируем стандартный вывод
	if cl.nextCmd != nil {
		ctx.stdOut = &buf
	}
	runCommand, ok := cmdMap[cl.cmd]
	if ok {
		err := runCommand(ctx)
		if err != nil {
			return err
		}
	} else {
		// Если команда отсутствует в стандартном наборе,
		// пытаемся запустить ее в настоящем shell.
		cmd := exec.Command(cl.cmd, strings.Split(cl.args, " ")...)
		if cl.args == "" {
			cmd.Args = nil
		}
		cmd.Stdin = ctx.stdin
		cmd.Stdout = ctx.stdOut
		err := cmd.Run()
		if err != nil {
			return err
		}
	}
	// если в пайпе есть команда, запускаем её
	// стандартный ввод берём из буфера
	if cl.nextCmd != nil {
		return cl.nextCmd.Execute(&buf, stdOut)
	}
	return nil
}

// parseCmdLine переводит строку, введенную пользователем в структуру команды
func parseCmdLine(c string) *cmdLine {
	c = strings.TrimSpace(c)
	pipe := strings.Split(c, "|")
	var nextCmd *cmdLine = nil
	if len(pipe) > 1 {
		c = pipe[0]
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
func echo(c cmdContext) error {
	fmt.Fprintln(c.stdOut, c.args)
	return nil
}

// ps выводит список процессов
func ps(c cmdContext) error {
	processes, err := gops.Processes()
	if err != nil {
		return err
	}
	fmt.Fprintln(c.stdOut, "PID\t\tname")
	fmt.Fprintln(c.stdOut, "------------------------")
	for _, p := range processes {
		fmt.Fprintf(c.stdOut, "%d\t\t%s\n", p.Pid(), p.Executable())
	}
	return nil
}

// pwd выводит текущий путь
func pwd(c cmdContext) error {
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	fmt.Fprintln(c.stdOut, dir)
	return nil
}

// kill убивает процесс с данным pid.
func kill(c cmdContext) error {
	pid, err := strconv.Atoi(c.args)
	if err != nil {
		return err
	}
	return syscall.Kill(pid, syscall.SIGINT)
}

func cd(c cmdContext) error {
	return os.Chdir(c.args)
}

// exit завершает работу оболочки.
func exit(cmdContext) error {
	return ErrExit
}

// parser обрабатывает строку, введенную пользователем.
func parser(c string) error {
	if c == "" || c == "\n" {
		return nil
	}
	cl := parseCmdLine(c)
	return cl.Execute(os.Stdin, os.Stdout)
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
