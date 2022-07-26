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

func downloadFile(w io.Writer, uri string) error {
	resp, err := http.Get(uri)
	if err != nil {
		return fmt.Errorf("http error: %w", err)
	}
	defer resp.Body.Close()
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

func downloadSite(sitePath, uri string, recLength int) error {
	resp, err := http.Get(uri)
	if err != nil {
		return fmt.Errorf("http error when downloading %q: %w", uri, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: status: %s", uri, resp.Status)
	}
	if !strings.Contains(resp.Header["Content-Type"][0], "text/html") {
		return fmt.Errorf("%s is not a web-site; Content-Type: %s", uri, resp.Header["Content-Type"])
	}
	if err := os.MkdirAll(sitePath, 0755); err != nil {
		return err
	}
	if err := os.Chdir(sitePath); err != nil {
		return err
	}

	f, err := os.Create("index.html")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	links, err := linkStripper(resp.Body, f)
	if err != nil {
		return err
	}
	log.Printf("%s saved to the file %s; links: %v", uri, path.Join(sitePath, "index.html"), links)

	return nil
}

func linkStripper(body io.Reader, file io.Writer) ([]string, error) {
	links := make([]string, 0)
	buf, err := io.ReadAll(body)
	if err != nil {
		return nil, err
	}
	for {
		link, before, buf, ok := nextLink(buf)
		if !ok {
			if err := write(file, buf); err != nil {
				return nil, err
			}
			break
		}
		links = append(links, link)
		before = append(before, []byte("ТУТ БУДЕТ НОВАЯ ССЫЛКА!")...)
		if err := write(file, before); err != nil {
			return nil, err
		}
	}
	return links, nil
}

func nextLink(p []byte) (link string, before, after []byte, ok bool) {
	const (
		href = `href="`
		src  = `src="`
	)
	s := string(p)
	hrefIdx := strings.Index(s, href)
	srcIdx := strings.Index(s, src)
	if hrefIdx == -1 && srcIdx == -1 {
		return "", nil, p, false
	}
	if srcIdx != -1 || srcIdx < hrefIdx {
		aaaaaaaa
		link, after, ok := getLink(s[srcIdx+len(src):])
		if !ok {
			return "", nil, p, false
		}
		return link, p[:srcIdx], after, true
	}
	link, after, ok = getLink(s[hrefIdx+len(href):])
	if !ok {
		return "", nil, p, false
	}
	return link, p[:srcIdx], after, true

}

func getLink(s string) (string, []byte, bool) {
	link, after, found := strings.Cut(s, "\"")
	if !found {
		return "", []byte(s), false
	}
	return link, []byte(after), true
}

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
		path := getPathName(*pathFlag, uri)
		if *recLength == 0 {
			*recLength = -1
		}
		if err := downloadSite(path, uri, *recLength); err != nil {
			log.Fatal(err)
		}
		return
	}
	filename := *outFile
	if filename == "" {
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
		return "./" + pathFlag
	}
	u, err := url.Parse(uri)
	if err != nil {
		log.Fatal(err)
	}
	return "./" + u.Hostname()
}
