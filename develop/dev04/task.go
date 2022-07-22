package main

import (
	_ "embed"
	"fmt"
	"sort"
	"strings"
	"unicode"
)

/*
=== Поиск анаграмм по словарю ===

Напишите функцию поиска всех множеств анаграмм по словарю.
Например:
'пятак', 'пятка' и 'тяпка' - принадлежат одному множеству,
'листок', 'слиток' и 'столик' - другому.

Входные данные для функции: ссылка на массив - каждый элемент которого - слово на русском языке в кодировке utf8.
Выходные данные: Ссылка на мапу множеств анаграмм.
Ключ - первое встретившееся в словаре слово из множества
Значение - ссылка на массив, каждый элемент которого, слово из множества. Массив должен быть отсортирован по возрастанию.
Множества из одного элемента не должны попасть в результат.
Все слова должны быть приведены к нижнему регистру.
В результате каждое слово должно встречаться только один раз.

Программа должна проходить все тесты. Код должен проходить проверки go vet и golint.
*/

//go:embed russian_nouns.txt
var text string

// var v = []string{
// 	"листок",
// 	"пятка",
// 	"слон",
// 	"тяпка",
// 	"Козёл",
// 	"слиток",
// 	"Тяпка",
// 	"столик",
// 	"Стол",
// 	"лост",
// 	"Пятак",
// 	"рот",
// 	"тор",
// }

// GetAnagrams составляет словарь анаграмм на основе переданного списка слов.
// В словарь попадают слова, имеющие хотя бы одну анаграмму в данном списке.
// Все слова переводятся в нижний регистр.
// Ключом является первое слово из множетсва данных анаграмм, встретившееся в переданном
// списке. Значением является отсортированный массив анаграмм к слову-ключу.
//
// Функция паникует, если какое-либо слово содержит что-то кроме букв и дефисов.
// По условию функция должна возвращать "ссылку на мапу". Поскольку мапа и так - ссылочноый
// тип, возвращаем её саму.
func GetAnagrams(wordsPtr *[]string) map[string][]string {
	// преобразуем всё в нижний регистр и оставим только уникальные слова
	words := onlyUniqueLowerCase(*wordsPtr)
	// построим карту анаграмм по инвариантным ключам
	keyMap := setKeyMap(words)

	anagramsMap := make(map[string][]string)
	for _, anagrams := range keyMap {
		// в результат включаем только слова, имеющие хотя бы одну анаграмму
		if len(anagrams) < 2 {
			continue
		}
		// ключ - первое слово из списка
		key := anagrams[0]
		anagrams = anagrams[1:]
		// сортируем массив анаграмм
		sort.Sort(sort.StringSlice(anagrams))
		anagramsMap[key] = anagrams
	}
	return anagramsMap
}

// letterKeyMap - структура данных для определения ключа, инвариантного для
// каждого набора анаграмм. Ключом является строка вида "<буква1><число букв1>..."
// Буквы даются в алфавитном порядке
type (
	letterKeyMap map[rune]int
	// letterKeySlice - массив для сортировки пар <буква> - <число букв>
	letterKeySlice [][2]string
)

// методы, реализующие интерфейс sort.Slice.
func (s letterKeySlice) Len() int           { return len(s) }
func (s letterKeySlice) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s letterKeySlice) Less(i, j int) bool { return s[i][0] < s[j][0] }

// String реализует интерфейс Stringer.
func (m letterKeyMap) String() string {
	slice := make(letterKeySlice, 0, len(m))
	for letter, numLetters := range m {
		slice = append(slice, [2]string{string(letter), fmt.Sprint(numLetters)})
	}
	sort.Sort(slice)
	var result strings.Builder
	for _, l := range slice {
		result.WriteString(fmt.Sprintf("%s%s", l[0], l[1]))
	}
	return result.String()
}

// getLetterKey возвращает ключ, общий для всех анаграмм данного слова.
// формат ключа "<буква1><число букв1 в слове>..." в алфавитном порядке.
// например "банан" --> "а2б1н2"
func getLetterKey(word string) string {
	lMap := make(letterKeyMap)
	for _, r := range strings.ToLower(word) {
		if !unicode.IsLetter(r) && r != '-' {
			panic(fmt.Sprintf("incorrect word: %s", strings.ToLower(word)))
		}
		lMap[r]++
	}
	return lMap.String()
}

// setKeyMap возвращает мапу всех множеств анаграмм по инвариантному ключу.
func setKeyMap(words []string) map[string][]string {
	keyMap := make(map[string][]string)
	for _, word := range words {
		key := getLetterKey(word)
		keyMap[key] = append(keyMap[key], word)
	}
	return keyMap
}

// onlyUniqueLowerCase убирает из массива повторяющиеся значения и приводит его к нижнему регистру.
func onlyUniqueLowerCase(arr []string) []string {
	set := make(map[string]bool)
	result := make([]string, 0, len(arr))
	for _, s := range arr {
		s := strings.ToLower(s)
		if !set[s] {
			result = append(result, s) // добавляем в результат только значения, которых ещё нет в map'е.
			set[s] = true
			continue
		}
	}
	return result
}

func main() {
	words := strings.Split(text, "\n")
	for key, anagrams := range GetAnagrams(&words) {
		fmt.Printf("%s:\n", key)
		for _, word := range anagrams {
			fmt.Printf("\t%s\n", word)
		}
	}
}
