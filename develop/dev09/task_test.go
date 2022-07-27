package main

import (
	"path"
	"strings"
	"testing"
)

func TestLinkStripper(t *testing.T) {
	r := strings.NewReader(`123<a href="http://link1.ru?qwe=12%20asd"></a>456<img src="/link2/image.jpg"></img>789<style src="/styles/link3.css">000`)
	w := strings.Builder{}
	links, err := copyWithRenewedLinks(r, &w, -1)
	if err != nil {
		t.Error(err)
	}
	t.Log("links:")
	for _, l := range links {
		t.Log(l)
	}
	t.Logf("out: %s", w.String())
}

func TestLinkToFileName(t *testing.T) {
	links := []string{
		"http://yandex.ru/qwe/iop/pic.jpg",
		"/this/that/css/style.css",
		"https://w3.org/qwerty?q=5&v=asd%20zxc"}
	for _, link := range links {
		t.Run(link, func(t *testing.T) {
			filePath, err := linkToFilePath(link)
			if err != nil {
				t.Error(err)
			}
			dir, fileName := path.Split(filePath)
			t.Logf("link: %s, dir: %s, fileName: %s", link, dir, fileName)

		})
	}

}
