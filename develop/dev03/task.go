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
		reverse      bool
		unique       bool
		numeric      bool
		ignoreSpaces bool
		k            kFlag
		lines        sort.StringSlice
	}

	kFlag []int
)

func (k *kFlag) String() string {
	return fmt.Sprintf("%v", *k)
}

func (k *kFlag) Set(s string) error {
	n, err := strconv.Atoi(s)
	if err != nil {
		return errors.New("flag must be an integer")
	}
	if n < 1 {
		return errors.New("value must be greater than or equal to 1")
	}
	*k = append(*k, n-1) // колонки в массиве нумеруются с 0
	return nil
}

func (s *GoSorter) Sort(lines []string) (string, error) {
	if s.unique {
		lines = onlyUnique(lines)
	}

	s.lines = sort.StringSlice(lines)
	if s.k != nil {
		if err := s.validateColumns(); err != nil {
			return "", err
		}
	}

	sort.Sort(s)
	return strings.Join(lines, "\n"), nil
}

func (s *GoSorter) validateColumns() error {
	c, err := s.getColumns()
	if err != nil {
		log.Fatalf("could not split file(s) to columns: %s", err)
	}
	for _, k := range s.k {
		if k+1 > len(c) {
			return fmt.Errorf("wrong value of -k flag: %d, file has %d columns", k+1, len(c))
		}
	}
	return nil
}

func (s *GoSorter) Len() int      { return s.lines.Len() }
func (s *GoSorter) Swap(i, j int) { s.lines.Swap(i, j) }
func (s *GoSorter) Less(i, j int) bool {
	if s.reverse {
		i, j = j, i
	}
	if s.k == nil && !s.numeric {
		return s.lines.Less(i, j)
	}
	if s.k == nil {
		return numericLess(s.lines, i, j)
	}
	return s.columnLess(s.lines, i, j)
}

func (s *GoSorter) columnLess(slice sort.StringSlice, i, j int) bool {
	lessFn := func(localSlice sort.StringSlice, i, j int) bool {
		return localSlice.Less(i, j)
	}
	if s.numeric {
		lessFn = numericLess
	}
	var isLess bool
	for _, k := range s.k {
		valI, _ := s.getColumnVal(i, k)
		valJ, _ := s.getColumnVal(j, k)
		isLess = lessFn(sort.StringSlice{valI, valJ}, 0, 1)
		if isLess {
			break
		}
		if lessFn(sort.StringSlice{valI, valJ}, 1, 0) {
			break
		}
	}
	return isLess
}

func numericLess(slice sort.StringSlice, i, j int) bool {
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
	numI, iHasNum := stripNumber(slice[i])
	numJ, jHasNum := stripNumber(slice[j])
	if !iHasNum && !jHasNum {
		return strings.Compare(slice[i], slice[j]) == -1
	}
	if iHasNum && jHasNum {
		return numI < numJ
	}
	return !iHasNum
}

func (s *GoSorter) getColumnVal(lineNum, colNum int) (string, error) {
	if lineNum >= len(s.lines) {
		return "", fmt.Errorf("line number %d out of range", lineNum)
	}
	fields := strings.Fields(s.lines[lineNum])
	if colNum >= len(fields) {
		return "", fmt.Errorf("column number %d out of range", colNum)
	}
	return fields[colNum], nil
}

func (s *GoSorter) getColumns() ([][]string, error) {
	numColumns := len(strings.Fields(s.lines[0]))
	// [<colNum>][<lineNum>]
	columns := make([][]string, numColumns)
	for i := range columns {
		columns[i] = make([]string, len(s.lines))
	}
	for lineNum, line := range s.lines {
		fields := strings.Fields(line)
		if len(fields) != numColumns {
			return nil, fmt.Errorf("wrong number of columns in the line: %q", line)
		}
		for colNum, field := range fields {
			columns[colNum][lineNum] = field
		}
	}
	return columns, nil
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
	flag.BoolVar(&s.ignoreSpaces, "b", false, "ignore spaces")
	flag.Var(&s.k, "k", "sort columns")
	flag.Parse()
	args := flag.Args()
	var lines []string
	if len(args) > 0 {
		lines, err = linesFromFiles(args)
	} else {
		lines, err = scanLines(os.Stdin)
	}
	if err != nil {
		log.Fatal(err)
	}
	sorted, err := s.Sort(lines)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(sorted)
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
