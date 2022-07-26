package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"unicode"
)

/*
=== Утилита cut ===

Принимает STDIN, разбивает по разделителю (TAB) на колонки, выводит запрошенные

Поддержать флаги:
-f - "fields" - выбрать поля (колонки)
-d - "delimiter" - использовать другой разделитель
-s - "separated" - только строки с разделителем

Программа должна проходить все тесты. Код должен проходить проверки go vet и golint.
*/

// Cut - разделитель на колонки.
type Cut struct {
	// номера колок для вывода
	// если среди номеров встречается -1, это означает, что нужно отобразить все колонки,
	// начиная с указанной в предыдущем элементе массива
	requiredColumns []int
	// разделитель колонок
	delimiter string
	// выводить только строки, в которых есть колонки
	separatedOnly bool
}

// Do разбивает строки, входящие из Stdin на колонки указанным разделителем
// и выводит запрошенные колонки (или их диапазон) в Stdout.
func (c Cut) Do() {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		// делим строки на поля
		fields := strings.Split(scanner.Text(), c.delimiter)
		// Если установлен флаг -s, и в этой строке нет разделения, пропуск.
		if c.separatedOnly && len(fields) <= 1 {
			continue
		}
		// если не указаны колонки - выводим всё
		if len(c.requiredColumns) == 0 {
			fmt.Fprintln(os.Stdout, scanner.Text())
			continue
		}
		resultLine := make([]string, 0, len(fields))
		for i, n := range c.requiredColumns {
			// -1 является служебным числом
			if n == -1 {
				continue
			}
			n-- // переводим номер колонки в индекс
			// если следующее число в массиве номеров колонок = -1 - выводим всеколонки,
			// начиная с данной
			if i < len(c.requiredColumns)-1 && c.requiredColumns[i+1] == -1 && n < len(fields) {
				resultLine = append(resultLine, fields[n:]...)
				continue
			}
			// если в строке есть колонка с таким номером, добавляем её к выводу
			if n < len(fields) {
				resultLine = append(resultLine, fields[n])
			}
		}
		fmt.Fprintln(os.Stdout, strings.Join(resultLine, c.delimiter))
	}
}

// NewCut создаёт новую структуру Cut и преобразует диапазон колонок, указанный
// во флаге -f в массив номеров.
func NewCut(reqFields string, delimeter string, separatedOnly bool) (*Cut, error) {
	requiredFields, err := parseNumFields(reqFields)
	if err != nil {
		return nil, err
	}
	return &Cut{
		requiredColumns: requiredFields,
		delimiter:       delimeter,
		separatedOnly:   separatedOnly,
	}, err
}

// parseNumFields преобразует строку в массив с номерами колонок
// формат строки: <range1>,<range2>...
//		где range может быть либо числом, либо диапазоном (1-3, 4-, -5).
func parseNumFields(s string) ([]int, error) {
	// не заморачиваемся с пустой строкой
	if s == "" {
		return []int{}, nil
	}
	res := make([]int, 0)
	// получаем поля
	fields := strings.Split(s, ",")
	for _, f := range fields {
		// проверяем, может быть в поле только число
		if !strings.Contains(f, "-") {
			n, err := strconv.Atoi(f)
			if err != nil {
				return nil, err
			}
			res = append(res, n)
			continue
		}
		// читаем первый элемент поля
		el, f, err := readElement(f)
		if err != nil {
			return nil, err
		}
		// если первый элемент "-":
		if el == "-" {
			if f == "" {
				return nil, fmt.Errorf("could not parse %q", s)
			}
			// второй элемент должен быть числом
			num, f, err := readElement(f)
			if err != nil {
				return nil, err
			}
			// больше элементов быть не должно
			if f != "" {
				return nil, fmt.Errorf("could not parse %q", s)
			}
			n, err := strconv.Atoi(num)
			if err != nil {
				return nil, err
			}
			// записываем диапазон
			for i := 1; i <= n; i++ {
				res = append(res, i)
			}
			continue
		}
		// остаётся вариант диапазона "n-m"
		n, err := strconv.Atoi(el)
		if err != nil {
			return nil, fmt.Errorf("could not parse %q", s)
		}
		// следующий элемент должен быть "-"
		el, f, err = readElement(f)
		if err != nil {
			return nil, err
		}
		if el != "-" {
			return nil, fmt.Errorf("unreachable: could not parse %q", s)
		}
		// и обязательно нужен третий элемент - число
		if f == "" {
			res = append(res, n, -1)
			continue
		}
		el, f, err = readElement(f)
		if err != nil {
			return nil, err
		}
		// получаем число
		m, err := strconv.Atoi(el)
		if err != nil {
			return nil, err
		}
		// больше элементов не должно быть
		if f != "" {
			return nil, fmt.Errorf("could not parse %q", s)
		}
		// если задан братный диапазон
		if n >= m {
			for i := n; i >= m; i-- {
				res = append(res, i)
			}
			continue
		}
		// добавляем диапазон в массив
		for i := n; i <= m; i++ {
			res = append(res, i)
		}
	}
	return res, nil
}

// readElement читает элемент из строки. Элементом может быть либо
// число, либо знак "-". Функция возвращает элемент, остаток строки (м.б. пустой)
// и ошибку.
func readElement(s string) (el, rest string, err error) {
	num := ""
	for i, r := range s {
		if r == '-' {
			if i == 0 {
				return "-", s[1:], nil
			}
			return num, s[i:], nil
		}
		if !unicode.IsDigit(r) {
			return "", "", fmt.Errorf("could not parse %q", s)
		}
		num += string(r)
	}
	return num, "", nil
}

func main() {
	fieldRange := flag.String("f", "", "fields to display")
	delimeter := flag.String("d", "\t", "delimeter")
	separatedOnly := flag.Bool("s", false, "display only lines that contain delimeter")
	flag.Parse()
	cut, err := NewCut(*fieldRange, *delimeter, *separatedOnly)
	if err != nil {
		flag.Usage()
		log.Fatal(err)
	}
	cut.Do()
}
