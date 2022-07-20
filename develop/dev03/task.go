package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

/*
=== Утилита sort ===

Отсортировать строки (man sort)
Основное

Поддержать ключи

-k — указание колонки для сортировки
-n — сортировать по числовому значению
-r — сортировать в обратном порядке
-u — не выводить повторяющиеся строки

Дополнительное

Поддержать ключи

-M — сортировать по названию месяца
-b — игнорировать хвостовые пробелы
-c — проверять отсортированы ли данные
-h — сортировать по числовому значению с учётом суффиксов

Программа должна проходить все тесты. Код должен проходить проверки go vet и golint.
*/

type (
	GoSorter struct {
		reverse bool
		unique  bool
		numeric bool
		k       kFlag
	}

	kFlag [][2]int
)

func (k *kFlag) String() string {
	res := make([]string, 0, len(*k))
	for _, col := range *k {
		res = append(res, fmt.Sprintf("[%d, %d]", col[0], col[1]))
	}
	return "{" + strings.Join(res, " ") + "}"
}

func (k *kFlag) Set(s string) error {
	fields := strings.Split(s, ",")
	if len(fields) != 2 {
		return errors.New("incorrect flag format")
	}
	a, err := strconv.Atoi(fields[0])
	if err != nil {
		return err
	}
	b, err := strconv.Atoi(fields[1])
	if err != nil {
		return err
	}
	*k = append(*k, [2]int{a, b})
	return nil
}

func (s *GoSorter) Sort(lines []string) string {
	var linesToSort sort.Interface
	if s.unique {
		lines = onlyUnique(lines)
	}
	linesToSort = sort.StringSlice(lines)
	if s.numeric {
		linesToSort = NumericStringSlice(lines)
	}
	if s.reverse {
		linesToSort = sort.Reverse(linesToSort)
	}
	sort.Sort(linesToSort)
	return strings.Join(lines, "\n")
}

type NumericStringSlice []string

func (x NumericStringSlice) Len() int {
	return len(x)
}

func (x NumericStringSlice) Swap(i, j int) {
	x[i], x[j] = x[j], x[i]
}

func (x NumericStringSlice) Less(i, j int) bool {
	stripNumber := func(s string) (int, bool) {
		if !unicode.IsDigit(rune(s[0])) {
			return -1, false
		}
		num := 0
		for _, r := range s {
			if !unicode.IsDigit(r) {
				break
			}
			digit := int(r - '0')
			num = num*10 + digit
		}
		return num, true
	}
	numI, iHasNum := stripNumber(x[i])
	numJ, jHasNum := stripNumber(x[j])
	if !iHasNum && !jHasNum {
		return strings.Compare(x[i], x[j]) == -1
	}
	if iHasNum && jHasNum {
		return numI < numJ
	}
	return !iHasNum
}

func onlyUnique(arr []string) []string {
	set := make(map[string]bool)
	result := make([]string, 0, len(arr))
	for _, s := range arr {
		if !set[s] {
			result = append(result, s)
			set[s] = true
			continue
		}
	}
	return result
}

func main() {
	var err error
	s := GoSorter{}
	flag.BoolVar(&s.reverse, "r", false, "reverse sorting")
	flag.BoolVar(&s.unique, "u", false, "show only first of an equal run")
	flag.BoolVar(&s.numeric, "n", false, "numeric sort")
	flag.Var(&s.k, "k", "sort columns")
	flag.Parse()
	args := flag.Args()
	log.Printf("sorter: %+v\nflag args: %v\n", s, args)
	var lines []string
	if len(args) > 0 {
		lines, err = linesFromFiles(args)
	} else {
		lines, err = scanLines(os.Stdin)
	}
	if err != nil {
		log.Fatal(err)
	}
	sorted := s.Sort(lines)
	fmt.Print(sorted)
}

func linesFromFiles(fileNames []string) ([]string, error) {
	lines := make([]string, 0)
	for _, name := range fileNames {
		f, err := os.Open(name)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		l, err := scanLines(f)
		if err != nil {
			return nil, err
		}
		lines = append(lines, l...)
	}
	return lines, nil
}

func scanLines(r io.Reader) ([]string, error) {
	lines := make([]string, 0)
	scanner := bufio.NewScanner(r)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return lines, nil
}
