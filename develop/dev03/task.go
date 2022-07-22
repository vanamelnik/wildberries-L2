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

// Пакет sort позволяет гибко настраивать сортировку чего бы то ни было путём имплементации интерфейса sort.Interface.
type (
	// goSorter содержит поля, влияющие на сортировку и управляет сортировкой через метод Less.
	goSorter struct {
		reverse bool  // обратная сортировка
		unique  bool  // показывать только уникальные значения
		numeric bool  // сортировка по числам
		k       kFlag // сортировка по колонкам
		lines   sort.StringSlice
	}

	// kFlag - тип, хранящий все флаги -k
	kFlag []int
)

// String реализует интерфейс flag.Value.
func (k *kFlag) String() string {
	return fmt.Sprintf("%v", *k)
}

// Set реализует интерфейс flag.Value.
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

// Sort производит настройку метода Less и сортировка массива строк с помощью sort.Sort.
func (s *goSorter) Sort(lines []string) (string, error) {
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
	return strings.Join(lines, "\n") + "\n", nil
}

// validateColumns проверяет, может ли файл быть разделённым на равное число колонок,
// и не превышают ли значения флагов -k числа колонок.
func (s *goSorter) validateColumns() error {
	numColumns := len(strings.Fields(s.lines[0]))
	for i, line := range s.lines {
		if len(strings.Fields(line)) != numColumns {
			return fmt.Errorf("could not split file(s) to columns: wrong number of fields in the line #%d", i)
		}
	}
	for _, k := range s.k {
		if k+1 > numColumns {
			return fmt.Errorf("wrong value of -k flag: %d, file has %d columns", k+1, numColumns)
		}
	}
	return nil
}

// Len наследуется от sort.StringSlice
func (s *goSorter) Len() int { return s.lines.Len() }

// Swap наследуется от sort.StringSlice
func (s *goSorter) Swap(i, j int) { s.lines.Swap(i, j) }

// Less реализует интерфейс sort.Interface. В зависимости от установленных флагов меняются параметры сортировки.
func (s *goSorter) Less(i, j int) bool {
	// для обратной сортировки меняем местами i и j.
	if s.reverse {
		i, j = j, i
	}
	// если не нужна сортировка по числовым значениям и по колонкам, наследуем метод sort.StringSLice.
	if s.k == nil && !s.numeric {
		return s.lines.Less(i, j)
	}
	// для числовой сортировки без разделения на колонки используем метод numericLess
	if s.k == nil {
		return numericLess(s.lines, i, j)
	}
	// иначе columnLess
	return s.columnLess(i, j)
}

// columnLess возвращает true, если значение поля в определённой колонке в строке i меньше, чем в строке j.
// Если значения равны, проверяются следующие колонки из массива k.
func (s *goSorter) columnLess(i, j int) bool {
	// lessFn - переопределяемая функция less в зависимости от метода сортировки (обычный или числовой)
	lessFn := func(localSlice sort.StringSlice, i, j int) bool {
		// для обычной сортировки наследуем метод sort.StringSlice.
		return localSlice.Less(i, j)
	}
	if s.numeric {
		// для числовой - используем метод numericLess.
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
		// проверяем, если valJ < valI, то можем подтвердить "not Less"
		if lessFn(sort.StringSlice{valI, valJ}, 1, 0) {
			break
		}
		// если valI >= valJ && valJ >= valI ==> valI == valJ; смотрим следующую колонку.
	}
	return isLess
}

// numericLess учитывает числовые значения. Поле, не содержащее числа считается меньшим поля, содержаего число.
func numericLess(slice sort.StringSlice, i, j int) bool {
	// stripNumber - внутренняя функция, пытающаяся извлечь число из начала строки
	stripNumber := func(s string) (int, bool) {
		if !unicode.IsDigit(rune(s[0])) {
			return -1, false
		}
		num := 0
		for _, r := range s {
			if !unicode.IsDigit(r) {
				break
			}
			digit := int(r - '0') //магия рун в действии!
			num = num*10 + digit
		}
		return num, true
	}

	numI, iHasNum := stripNumber(slice[i])
	numJ, jHasNum := stripNumber(slice[j])

	// если оба поля не содержат чисел, наследуем метод sort.StringSlice
	if !iHasNum && !jHasNum {
		return slice.Less(i, j)
	}
	// если оба поля содержат числа, возвращаем их сравнение.
	if iHasNum && jHasNum {
		return numI < numJ
	}
	// одно из полей содержит число, другое - нет. Поле, не содержащее число считается меньшим.
	return !iHasNum
}

// getColumnVal возвращает значение поля с заданными номерами строки и колонки.
func (s *goSorter) getColumnVal(lineNum, colNum int) (string, error) {
	if lineNum >= len(s.lines) {
		return "", fmt.Errorf("line number %d out of range", lineNum)
	}
	fields := strings.Fields(s.lines[lineNum])
	if colNum >= len(fields) {
		return "", fmt.Errorf("column number %d out of range", colNum)
	}
	return fields[colNum], nil
}

// onlyUnique убирает из массива повторяющиеся значения.
func onlyUnique(arr []string) []string {
	set := make(map[string]bool)
	result := make([]string, 0, len(arr))
	for _, s := range arr {
		if !set[s] {
			result = append(result, s) // добавляем в результат только значения, которых ещё нет в map'е.
			set[s] = true
			continue
		}
	}
	return result
}

func main() {
	var err error
	// устанавливаем флаги в структуре GoSorter
	s := goSorter{}
	flag.BoolVar(&s.reverse, "r", false, "reverse sorting")
	flag.BoolVar(&s.unique, "u", false, "show only first of an equal run")
	flag.BoolVar(&s.numeric, "n", false, "numeric sort")
	flag.Var(&s.k, "k", "sort columns")
	flag.Parse()

	args := flag.Args()
	var lines []string
	if len(args) > 0 {
		// если указаны файлы, читаем их
		lines, err = linesFromFiles(args)
	} else {
		// иначе читаем из stdin
		lines, err = scanLines(os.Stdin)
	}
	if err != nil {
		log.Fatal(err)
	}
	sorted, err := s.Sort(lines)
	if err != nil {
		log.Fatal(err)
	}
	// выводим результат в stdout
	fmt.Println(sorted)
}

// linesFromFiles читает файлы, имена которых переданы в массиве, и возвращает их объединенное
// содержимое в виде массива строк.
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

// scanLines читает построчно переданный reader.
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
