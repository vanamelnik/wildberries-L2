package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
)

/*
=== Утилита wget ===

Реализовать утилиту wget с возможностью скачивать сайты целиком

Программа должна проходить все тесты. Код должен проходить проверки go vet и golint.
*/

type WGet struct {
	writer io.Writer
}

func (w WGet) Do(uri string) error {
	resp, err := http.Get(uri)
	if err != nil {
		return fmt.Errorf("http error: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%s: %s", resp.Status, string(body))
	}
	if w.writer != nil {
		_, err := io.Copy(w.writer, resp.Body)
		if err != nil {
			return err
		}
	}
	return nil
}

func main() {
	flag.Parse()
	uri := flag.Arg(0)
	WGet{os.Stdout}.Do(uri)
}
