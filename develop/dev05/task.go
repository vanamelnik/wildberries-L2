package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strings"
)

/*
=== Утилита grep ===

Реализовать утилиту фильтрации (man grep)

Поддержать флаги:
-A - "after" печатать +N строк после совпадения
-B - "before" печатать +N строк до совпадения
-C - "context" (A+B) печатать ±N строк вокруг совпадения
-c - "count" (количество строк)
-i - "ignore-case" (игнорировать регистр)
-v - "invert" (вместо совпадения, исключать)
-F - "fixed", точное совпадение со строкой, не паттерн
-n - "line num", печатать номер строки

Программа должна проходить все тесты. Код должен проходить проверки go vet и golint.
*/

// Grep - фильтр по шаблону.
type Grep struct {
	// allLines хранит все строки, подающиеся на фильтр
	allLines []string

	// паттерны (регулярный и строковый)
	pattern    *regexp.Regexp
	strPattern string

	// управление контекстом отображения
	after   uint
	before  uint
	context uint

	// флаги фильтра
	printLineNum    bool
	printFileName   bool
	ignoreCase      bool
	printLinesCount bool
	fixed           bool
	invertMatch     bool
}

// SetPattern устанавливает шаблон для фильтра в зависимости от установленных флагов.
func (g *Grep) SetPattern(p string) (err error) {
	if g.ignoreCase {
		p = strings.ToLower(p)
	}
	// если установлен флаг -F - используем строковый шаблон
	if g.fixed {
		g.strPattern = p
		return nil
	}
	g.pattern, err = regexp.Compile(p)
	return err
}

// setContext - вспомогательная функция, рассчитывающая параметры before и after
// в зависимости от флага -C. Если установлен флаг -c - флаги контекста игнорируются.
func (g *Grep) setContext() {
	if g.printLinesCount {
		g.printLineNum = false
		g.after, g.before, g.context = 0, 0, 0
		return
	}
	// before и after не должны быть меньше, чем значение флага -C
	if g.context > g.after {
		g.after = g.context
	}
	if g.context > g.before {
		g.before = g.context
	}
}

// Do выводит строки содержащие паттерн, установленный в структуре.
func (g *Grep) Do(r grepReadCloser) {
	g.setContext()
	scanner := bufio.NewScanner(r)
	scanner.Split(bufio.ScanLines)
	g.allLines = make([]string, 0)
	// goodLines хранит индексы строк, совпадающих с шаблоном.
	goodLines := make([]int, 0)

	for i := 0; scanner.Scan(); i++ {
		line := scanner.Text()
		g.allLines = append(g.allLines, line)
		if g.ignoreCase {
			line = strings.ToLower(line)
		}
		// в зависимости от установленных флагов устанавливаем условие совпадения
		var matchCase bool
		if g.fixed { // для строковых шаблонов
			matchCase = g.strPattern == line
		} else { // для регулярных шаблонов
			matchCase = g.pattern.MatchString(line)
		}
		// если нужно инвертировать вывод - инвертируем условие
		if g.invertMatch {
			matchCase = !matchCase
		}
		// собственно, момент фильтрации
		if matchCase {
			goodLines = append(goodLines, i)
		}
	}

	// если нам нужно вывести только число строк
	if g.printLinesCount {
		g.printLine(fmt.Sprint(len(goodLines)), -1, r)
		return
	}

	// расширяем область выводимых строк засчёт контекста
	goodLines = g.getLinesWithContext(goodLines)

	// выводим строки в Stdout
	for _, n := range goodLines {
		g.printLine(g.allLines[n], n, r)
	}
}

// getLinesWithContext расширяет набор индексов выводимых строк
// в соответствии с установленными переменными контеста before и after.
func (g *Grep) getLinesWithContext(lineNums []int) []int {
	result := make([]int, 0)
	for _, n := range lineNums {
		// определяем интервал выводимых строк в соответствии с контекстом и границами
		before := n - int(g.before)
		if before < 0 {
			before = 0
		}
		after := n + int(g.after)
		if after >= len(g.allLines) {
			after = len(g.allLines) - 1
		}
		// добавляем индексы выводимых строк в результирующий массив
		for i := before; i <= after; i++ {
			if len(result) > 1 && result[len(result)-1] >= i {
				// если данные строки уже есть - пропускаем
				continue
			}
			result = append(result, i)
		}
	}
	return result
}

// printLine выводит строку в Stdout. При необходимости к строке добавляется
// имя файла и номер строки.
func (g *Grep) printLine(line string, lineNum int, r grepReadCloser) {
	if g.printLineNum {
		line = fmt.Sprintf("%d:%s", lineNum, line)
	}
	if g.printFileName {
		line = fmt.Sprintf("%s:%s", r.fileName, line)
	}
	fmt.Fprintln(os.Stdout, line)
}

func main() {
	g := Grep{}
	var err error
	// устанавливаем флаги
	flag.UintVar(&g.after, "A", 0, "print +N lines after")
	flag.UintVar(&g.before, "B", 0, "print +N lines before")
	flag.UintVar(&g.context, "C", 0, "print ±N lines before and after")
	flag.BoolVar(&g.printLineNum, "n", false, "print line number")
	flag.BoolVar(&g.ignoreCase, "i", false, "ignore case")
	flag.BoolVar(&g.printLinesCount, "c", false, "print number of matching lines")
	flag.BoolVar(&g.fixed, "F", false, "pattern is a string")
	flag.BoolVar(&g.invertMatch, "v", false, "select non-matching lines")
	flag.Parse()

	// первый параметр - шаблон
	p := flag.Arg(0)
	if p == "" {
		flag.Usage()
		os.Exit(1)
	}
	if err := g.SetPattern(p); err != nil {
		flag.Usage()
		log.Fatalf("incorrect pattern: %s: %s", p, err)
	}

	// открываем источники данных
	grepReaders, err := getReaders()
	if err != nil {
		flag.Usage()
		log.Fatal(err)
	}
	if len(grepReaders) > 1 {
		// если информация из нескольких файлов - добавляем имя файла к выводу
		g.printFileName = true
	}
	for _, r := range grepReaders {
		g.Do(r)
		r.Close()
	}
}

// grepReadCloser - ридер, содержащий имя файла (которое, возможно, понадобится вывести
// вместе с результатом)
type grepReadCloser struct {
	fileName string
	reader   io.ReadCloser
}

func (r grepReadCloser) Read(p []byte) (n int, err error) {
	return r.reader.Read(p)
}
func (r grepReadCloser) Close() error {
	return r.reader.Close()
}

// getReaders открывает переданные в аргументах в файлы и возвращает их в виде
// grepReader'ов. В случае отсутствия в аргументах имен файлов, возвращается
// массив с единственным ридером, указывающим на Stdin.
func getReaders() ([]grepReadCloser, error) {
	if len(flag.Args()) < 2 {
		return []grepReadCloser{
			{reader: os.Stdin},
		}, nil
	}
	readers := make([]grepReadCloser, 0, len(flag.Args())-1)
	for _, name := range flag.Args()[1:] {
		f, err := os.Open(name)
		if err != nil {
			return nil, fmt.Errorf("could not open file %s: %s", name, err)
		}
		readers = append(readers, grepReadCloser{fileName: name, reader: f})
	}
	return readers, nil
}
