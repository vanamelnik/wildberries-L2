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
		lines   sort.Interface
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

	if s.reverse {
		s.lines = sort.Reverse(s.lines)
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
	if s.k == nil && !s.numeric {
		return s.lines.Less(i, j)
	}
	lines, ok := s.lines.(sort.StringSlice)
	if !ok {
		panic("unreachable - sorting interface is not a string slice")
	}
	if s.k == nil {
		return numericLess(lines, i, j)
	}
	return s.columnLess(lines, i, j)
}

func (s *GoSorter) columnLess(slice sort.StringSlice, i, j int) bool {
	lessFn := func(localSlice sort.StringSlice, i, j int) bool {
		return localSlice.Less(i, j)
	}
	if s.numeric {
		lessFn = numericLess
	}
	//nolint: errcheck
	columns, _ := s.getColumns()
	var isLess bool
	for _, k := range s.k {
		column := columns[k]
		isLess = lessFn(column, i, j)
		log.Printf("column: %v\n %s < %s == %v\n", column, column[i], column[j], isLess)
		if isLess {
			break
		}
		if lessFn(column, j, i) {
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

func (s *GoSorter) getColumns() ([][]string, error) {
	lines, ok := s.lines.(sort.StringSlice)
	if !ok {
		panic("unreachable - sorting interface is not a string slice")
	}
	numColumns := len(strings.Fields(lines[0]))
	// [<colNum>][<lineNum>]
	columns := make([][]string, numColumns)
	for i := range columns {
		columns[i] = make([]string, len(lines))
	}
	for lineNum, line := range lines {
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
	sorted, err := s.Sort(lines)
	if err != nil {
		log.Fatal(err)
	}
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
