package main

import (
	"fmt"
	"math/rand"
	"os/exec"
	"strings"
	"testing"

	"github.com/bxcodec/faker/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// для тестирования используем сравнение с результатами, выдаваемыми оригинальной утилитой sort.
func TestSort(t *testing.T) {
	const numLines = 12 // для теста на уникальные значения число строк должно быть кратным 4.
	const numWords = 5
	type args struct {
		reverse bool
		unique  bool
		numeric bool
		columns []int
	}
	tests := []struct {
		name   string
		argStr []string
		args   args
	}{
		{
			name:   "Normal",
			argStr: nil,
			args:   args{columns: nil},
		},
		{
			name:   "Reverse",
			argStr: []string{"-r"},
			args:   args{reverse: true, columns: nil},
		},
		{
			name:   "Numeric",
			argStr: []string{"-n"},
			args:   args{numeric: true, columns: nil},
		},
		{
			name:   "Numeric reverse",
			argStr: []string{"-n", "-r"},
			args:   args{numeric: true, reverse: true, columns: nil},
		},
		{
			name:   "Columns",
			argStr: []string{"-k 2, 4"},
			args:   args{columns: []int{2 - 1, 4 - 1}},
		},
		{
			name:   "Columns numeric",
			argStr: []string{"-k 1", "-n"},
			args:   args{numeric: true, columns: []int{1 - 1}},
		},
		{
			name:   "Columns reverse",
			argStr: []string{"-k 3r"},
			args:   args{reverse: true, columns: []int{3 - 1}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lines := randomLines(numLines, numWords)
			cmd := exec.Command("sort", tt.argStr...)
			in, err := cmd.StdinPipe()
			require.NoError(t, err)
			fmt.Fprint(in, strings.Join(lines, "\n"))
			in.Close()

			want, err := cmd.Output()
			require.NoError(t, err)
			got, err := (&goSorter{
				reverse: tt.args.reverse,
				numeric: tt.args.numeric,
				k:       tt.args.columns,
			}).Sort(lines)
			assert.NoError(t, err)
			assert.Equal(t, string(want), got)
		})
	}
	t.Run("Unique", func(t *testing.T) {
		lines := randomLines(numLines/2, numWords)
		lines1 := randomLines(numLines/4, numWords)
		lines = append(lines1, lines...)
		lines = append(lines, lines1...)
		cmd := exec.Command("sort", "-u")
		in, err := cmd.StdinPipe()
		require.NoError(t, err)
		fmt.Fprint(in, strings.Join(lines, "\n"))
		in.Close()
		want, err := cmd.CombinedOutput()
		require.NoError(t, err)
		got, err := (&goSorter{unique: true}).Sort(lines)
		assert.NoError(t, err)
		assert.Equal(t, numLines*3/4+1, len(strings.Split(got, "\n")))
		assert.Equal(t, string(want), got)

	})
}

func TestSortError(t *testing.T) {
	t.Run("Wrong columns", func(t *testing.T) {
		testLines := `qwe wer ert
		asd sdf dfg
		zxc xcv cvb bnm
		asd sdf dfg`
		_, err := (&goSorter{k: []int{2}}).Sort(strings.Split(testLines, "\n"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "wrong number of fields in the line")
	})
	t.Run("k-flag out of range", func(t *testing.T) {
		testLines := `qwe wer ert
		asd sdf dfg
		zxc xcv cvb
		asd sdf dfg`
		_, err := (&goSorter{k: []int{2, 8}}).Sort(strings.Split(testLines, "\n"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "wrong value of -k flag")
	})
}

// randomLines возвращает массив строк с рандомными словами и числами.
func randomLines(numLines, numWords int) []string {
	result := make([]string, 0, numLines)
	for i := 0; i < numLines; i++ {
		sentence := make([]string, numWords)
		for i := range sentence {
			sentence[i] = faker.Word()
		}
		result = append(result, fmt.Sprintf("%d. %s", rand.Intn(1000), strings.Join(sentence, " ")))
	}
	return result
}
