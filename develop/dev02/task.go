package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

/*
=== Задача на распаковку ===

Создать Go функцию, осуществляющую примитивную распаковку строки, содержащую повторяющиеся символы / руны, например:
	- "a4bc2d5e" => "aaaabccddddde"
	- "abcd" => "abcd"
	- "45" => "" (некорректная строка)
	- "" => ""
Дополнительное задание: поддержка escape - последовательностей
	- qwe\4\5 => qwe45 (*)
	- qwe\45 => qwe44444 (*)
	- qwe\\5 => qwe\\\\\ (*)

В случае если была передана некорректная строка функция должна возвращать ошибку. Написать unit-тесты.

Функция должна проходить все тесты. Код должен проходить проверки go vet и golint.
*/

// ErrIncorrectString - ошибка, возвращаемая в случае, если формат строки некорректен.
var ErrIncorrectString = errors.New("incorrect string")

// Unpack распаковывает строку в соотвтетсвии с форматом, указанном в условии.
func Unpack(s string) (string, error) {
	res := strings.Builder{}
	runes := []rune(s)
	processingLetter := true // флаг, указывающий, на то, что в данный момент обрабатывается буква.
	var currentLetter rune
	for i := 0; i < len(runes); i++ { // не используем range ради гибкого использования i.
		r := runes[i]
		count := 1 // счётчик букв по умолчанию равен 1
		// clojure для сохранения результата в итоговую строку
		saveResult := func() { res.WriteString(strings.Repeat(string(currentLetter), count)) }

		// если сейчас должна обрабатываться буква
		if processingLetter {
			currentLetter = r
			if !unicode.IsLetter(r) { // не буква?
				if r != '\\' { // проверяем, нет ли здесь начала escape-последовательности
					return "", ErrIncorrectString
				}
				if i == len(runes)-1 {
					return "", ErrIncorrectString
				}
				// если escape - делаем следующую руну сохраняемой в результат, какой бы она ни была
				i++
				currentLetter = runes[i]
			}
			// если это ещё не конец, а следующая руна - цифра:
			if i < len(runes)-1 && (unicode.IsDigit(runes[i+1]) && runes[i+1] != '\\') {
				processingLetter = false
			} else {
				// следующий символ - не цифра, либо текущая руна - последняя в строке -
				// значит записываем результат
				saveResult()
			}
			continue // место буквы обработано, продолжаем цикл
		}
		// если мы здесь, то руна должна быть цифрой
		if !unicode.IsDigit(r) {
			return "", ErrIncorrectString
		}
		var err error
		count, err = strconv.Atoi(string(r))
		if err != nil { // мы только что проверили, что r - цифра...
			panic(fmt.Sprintf("unreachable: could not convert %s to numeric", string(r)))
		}
		// currentLetter и count заданы, сохраняем результат
		saveResult()
		// за цифрой должна идти буква
		processingLetter = true
	}
	return res.String(), nil
}

func main() {
	fmt.Println(Unpack("a4bc2d5e"))
	fmt.Println(Unpack("abcd"))
	fmt.Println(Unpack("45"))
	fmt.Println(Unpack(""))
	fmt.Println(Unpack(`qwe\4\5`))
	fmt.Println(Unpack(`qwe\45`))
	fmt.Println(Unpack(`qwe\\5`))

}
