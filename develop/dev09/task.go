package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
)

/*
=== Утилита wget ===

Реализовать утилиту wget с возможностью скачивать сайты целиком

Программа должна проходить все тесты. Код должен проходить проверки go vet и golint.
*/

// downloadFile скачивает данные по ссылке и перенаправляет их в данный writer.
func downloadFile(w io.Writer, uri string) error {
	resp, err := http.Get(uri)
	if err != nil {
		return fmt.Errorf("http error: %w", err)
	}
	defer resp.Body.Close()
	// проверяем статус
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status: %s", resp.Status)
	}
	if w != nil {
		_, err := io.Copy(w, resp.Body)
		if err != nil {
			return err
		}
	}
	return nil
}

// downloadSite рекурсивно скачивает сайт в указанную директорию
// если recLevel = -1 - происходит скачивание до победного конца.
func downloadSite(filePath, parentPath, uri string, recLevel int) error {
	if recLevel == 0 {
		return nil
	}
	// посылаем запрос к странице
	resp, err := http.Get(uri)
	if err != nil {
		return fmt.Errorf("http error when downloading %q: %w", uri, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: status: %s", uri, resp.Status)
	}
	// создаём поддиректорию в родительской
	dir := path.Join(parentPath, path.Dir(filePath))
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	// создаём файл для данной страницы
	f, err := os.Create(path.Join(parentPath, filePath))
	if err != nil {
		return err
	}
	defer f.Close()
	defer resp.Body.Close()
	// копируем страницу, попутно выдирая ссылки и заменяя их на пути к файлам
	// TODO: реализовать мапу со скачанными ссылками, чтобы не скачивать несколько раз и
	// избежать рекурсии
	links, err := copyWithRenewedLinks(resp.Body, f, recLevel)
	if err != nil {
		return err
	}
	log.Printf("recursion level: %d, %s saved to the file %s\n\n", recLevel, uri, filePath)
	if recLevel != 1 {
		// далее скачиваем все найденные ссылки
		links := unique(links)
		for _, link := range links {
			fPath, err := linkToFilePath(link)
			if err != nil {
				log.Printf("link: %s, error: %s\n\n", link, err)
			}
			rl := recLevel
			if recLevel != -1 {
				rl--
			}
			// если ссылкаотносительная, приделываем её к базовому URL
			u, err := url.Parse(link)
			if err != nil {
				log.Printf("link: %s, error: %s\n\n", link, err)
			}
			if !u.IsAbs() {
				uParent, err := url.Parse(uri)
				if err != nil {
					log.Printf("link: %s, error: %s\n\n", link, err)
				}
				link = uParent.ResolveReference(u).String()
			}
			err = downloadSite(path.Join(parentPath, fPath), path.Dir(filePath), link, rl)
			if err != nil {
				log.Printf("filePath: %s, link% s, error: %s\n\n", fPath, link, err)
			}
		}
	}
	return nil
}

// unique удаляет из слайса повторы.
func unique(ss []string) []string {
	res := make([]string, 0, len(ss))
	m := make(map[string]bool)
	for _, s := range ss {
		if !m[s] {
			res = append(res, s)
		}
	}
	return res
}

// linkToFilePath преобразует URL в имя файла с путём.
func linkToFilePath(link string) (string, error) {
	u, err := url.Parse(link)
	if err != nil {
		return "", err
	}
	p := u.Hostname() + "/" + u.EscapedPath()
	fullPath := strings.Split(p, "/")
	if len(fullPath[len(fullPath)-1]) == 0 {
		fullPath = fullPath[:len(fullPath)-1]
	}
	return path.Join(fullPath...), nil
}

// copyWithRenewedLinks копирует данные из reader'а во writer.
// все найденные ссылки возвращаются в виде массива, а в копии -
// заменяются на пути к локальным файлам (если уровень рекурсии не 1).
func copyWithRenewedLinks(body io.Reader, file io.Writer, recLevel int) ([]string, error) {
	if recLevel == 1 {
		_, err := io.Copy(file, body)
		if err != nil {
			return nil, err
		}
		return []string{}, nil
	}
	links := make([]string, 0)
	buf, err := io.ReadAll(body)
	if err != nil {
		return nil, err
	}
	var link string
	var before []byte
	var ok bool
	for {
		// находим ссылку
		link, before, buf, ok = nextLink(buf)
		if !ok {
			// если не нашли - записываем остаток
			if err := write(file, buf); err != nil {
				return nil, err
			}
			break
		}
		links = append(links, link)
		filePath, err := linkToFilePath(link)
		if err != nil {
			return nil, err
		}
		// заменяем ссылку на путь к файлу
		before = append(before, []byte(filePath)...)
		if err := write(file, before); err != nil {
			return nil, err
		}
	}
	return links, nil
}

// nextLink находит первую ссылку в строке (по ключевым словам "href=" и "src=")
// и возвращает её, и фрагменты перед ссылкой и после неё.
func nextLink(p []byte) (link string, before, after []byte, ok bool) {
	const (
		href = `href="`
		src  = `src="`
	)
	s := string(p)
	hrefIdx := strings.Index(s, href)
	srcIdx := strings.Index(s, src)
	if hrefIdx == -1 && srcIdx == -1 {
		// ссылка ненайдена
		return "", nil, p, false
	}
	// первым попался href?
	if (hrefIdx != -1 && hrefIdx < srcIdx) || srcIdx == -1 {
		// достаём выражение в кавычках
		link, after, ok := getLink(s[hrefIdx+len(href):])
		if !ok {
			return "", nil, p, false
		}
		return strings.TrimSpace(link), p[:hrefIdx+len(href)], after, true
	}
	// если ссылка по ключевому слову "src="
	link, after, ok = getLink(s[srcIdx+len(src):])
	if !ok {
		return "", nil, p, false
	}
	return strings.TrimSpace(link), p[:srcIdx+len(src)], after, true
}

// getLink возвращает область строки до символа кавычки и остаток строки.
func getLink(s string) (string, []byte, bool) {
	parentIdx := strings.IndexByte(s, '"')
	if parentIdx == -1 {
		return "", []byte(s), false
	}
	link := s[:parentIdx]
	after := []byte(s[parentIdx:])
	return link, []byte(after), true
}

// write - вспомогательная функция для записи во writer.
func write(w io.Writer, p []byte) error {
	if n, err := w.Write(p); err != nil || n < len(p) {
		return fmt.Errorf("could not write to file: error: %v, n: %d, buf: %d", err, n, len(p))
	}
	return nil
}

func main() {
	outFile := flag.String("O", "", "output file name")
	mirror := flag.Bool("m", false, "download whole site")
	recLength := flag.Int("r", 0, "the length of recursion, 0 = infinity")
	pathFlag := flag.String("P", "", "path name for site downloading")

	flag.Parse()

	uri := flag.Arg(0)
	if uri == "" {
		flag.Usage()
		os.Exit(1)
	}
	if *mirror {
		// если установлен флаг -m - скачиваем сайт целиком
		fPath := path.Join(getPathName(*pathFlag, uri), "index.html")
		if *recLength == 0 {
			*recLength = -1
		}
		if err := downloadSite(fPath, ".", uri, *recLength); err != nil {
			log.Fatal(err)
		}
		return
	}
	// берем имя файла либо из флага...
	filename := *outFile
	if filename == "" {
		// либо генерируем из URL
		filename = fileNameFromURI(uri)
		log.Println("filename from URI:", filename)
	}
	f, err := os.Create(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	if err := downloadFile(f, uri); err != nil {
		log.Fatal(err)
	}
}

func fileNameFromURI(uri string) string {
	fields := strings.Split(uri, "/")
	name := fields[len(fields)-1]
	if name == "" {
		name = fields[len(fields)-2]
	}
	return name
}

func getPathName(pathFlag, uri string) string {
	if pathFlag != "" {
		return path.Join(".", pathFlag)
	}
	u, err := url.Parse(uri)
	if err != nil {
		log.Fatal(err)
	}
	return path.Join(".", u.Hostname())
}
