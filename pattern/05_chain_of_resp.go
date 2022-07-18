package pattern

import "fmt"

/*
	Реализовать паттерн «цепочка вызовов».
Объяснить применимость паттерна, его плюсы и минусы, а также реальные примеры использования данного примера на практике.
	https://en.wikipedia.org/wiki/Chain-of-responsibility_pattern
*/

// Паттерн "цепочка вызовов" используется для создания конвеера, каждое звено которого делает
// свою часть работы, не влияя на состояние других звеньев.

// интерфейс процессора - звена цепочки
type processor interface {
	process(request)
}

// интерфейс цепочки обязанностей (сама является процессором)
type cor interface {
	add(processor)
	processor
}

// реализация цепочки
type chain []processor

func (c *chain) add(p ...processor) {
	*c = append(*c, p...)
}

func (c *chain) process(r request) {
	for _, p := range *c {
		p.process(r)
	}
}

type kind int

const (
	urgent kind = 1 << iota
	special
	valuable
)

type request struct {
	kind kind
	data string
}

// реализация процессоров

// printer выводит данные в запроса консоль
type printer struct{}

func (p *printer) process(r request) {
	fmt.Printf("\nNew request: %s\n", r.data)
}

// saver сохраняет запрос, если его kind - valuable или special.
type saver struct{}

func (s *saver) process(r request) {
	// обрабатывает не все запросы
	if r.kind&(valuable|special) != 0 {
		fmt.Printf("Save request: %s\n", r.data)
		// save request
	}
}

// logger выводит в консоль данные, если kind является urgent.
type logger struct{}

func (l *logger) process(r request) {
	if r.kind&urgent != 0 {
		fmt.Printf("Log request: %s\n", r.data)
		// log request
	}
}

// клиентский код
func corDemo() {
	p := new(printer)
	l := new(logger)
	s := new(saver)
	c := make(chain, 0)
	c.add(p, l, s)
	r := request{0, "Average"}
	c.process(r)
	r = request{valuable, "Do not forget"}
	c.process(r)
	r = request{urgent | special, "Alert!!!"}
	c.process(r)
}
