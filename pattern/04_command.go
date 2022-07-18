package pattern

import (
	"fmt"
	"strings"
	"time"
)

/*
	Реализовать паттерн «комманда».
Объяснить применимость паттерна, его плюсы и минусы, а также реальные примеры использования данного примера на практике.
	https://en.wikipedia.org/wiki/Command_pattern
*/

// Поведенческий паттерн "Команда" посволяет превратить операции в объекты и передавать их как аргументы.
// Это позволяет ставить запросы в очередь, осуществлять отложенный запуск, логировать и отменять.
// В нижеприведенном примере реализован интерфейс умного дома (receiver), принимающий команды включить/выключить
// свет и сварить кофе. Также реализован планировщик, способный принимать команды и выполнять их по расписанию
// (команды могут быть в т.ч. и к другим исполнителям).

// интерфейс команды
type command interface {
	execute()
}

// интерфейс исполнителя
type receiver interface {
	turnLightsOn()
	turnLightsOff()
	makeCoffee()
}

// конкретная команда: включить свет
type cmdTurnLightsOn struct {
	r receiver
}

func (c cmdTurnLightsOn) execute() {
	c.r.turnLightsOn()
}

// конкретная команда: выключить свет
type cmdTurnLightsOff struct {
	r receiver
}

func (c cmdTurnLightsOff) execute() {
	c.r.turnLightsOff()
}

// конкретная команда: приготовить кофе
type cmdMakeCoffee struct {
	r receiver
}

func (c cmdMakeCoffee) execute() {
	c.r.makeCoffee()
}

// конкретный receiver - умный дом
type smartHouse struct {
	lightsOn    bool
	coffeeReady bool
}

// действия исполнителя
func (s *smartHouse) turnLightsOn() {
	fmt.Println("Включаем свет")
	s.lightsOn = true
}
func (s *smartHouse) turnLightsOff() {
	fmt.Println("Выключаем свет")
	s.lightsOn = false
}
func (s *smartHouse) makeCoffee() {
	fmt.Print("Готовим кофе... ")
	time.Sleep(time.Second * 3)
	s.coffeeReady = true
	fmt.Println(" выполнено!")
}
func (s *smartHouse) printState() {
	sb := strings.Builder{}
	sb.WriteString("Состояние умного дома:\n\t- дом стоит\n")
	if s.lightsOn {
		sb.WriteString("\t- свет горит\n")
	} else {
		sb.WriteString("\t- свет не горит\n")
	}
	if s.coffeeReady {
		sb.WriteString("\t- кофе готов\n")
	} else {
		sb.WriteString("\t- кофе нет\n")
	}
	fmt.Println(sb.String())
}

// планировщик, принимающий команды на выполнение в определенное время
type (
	scheduler struct {
		taskCh chan task
	}

	task struct {
		at  time.Time
		cmd command
	}
)

func newScheduler() *scheduler {
	s := scheduler{
		taskCh: make(chan task, 1),
	}
	// воркер, принимающий задания
	go func() {
		fmt.Println("Планировщик работает")
		for t := range s.taskCh {
			// выполнить задачу
			time.AfterFunc(t.at.Sub(time.Now()), t.cmd.execute)
		}
		fmt.Println("Планировщик остановлен, но таймеры могут сработать")
	}()
	return &s
}

// добавить задание
func (s *scheduler) addTask(t time.Time, c command) {
	s.taskCh <- task{
		at:  t,
		cmd: c,
	}
}

// остановить планировщик (в этой реализации задания остаются в очереди
// после остановки планировщика, и будут исполнены, если выполнение программы не прекратится раньше)
func (s *scheduler) stop() {
	if s.taskCh != nil {
		close(s.taskCh)
		s.taskCh = nil
	}
}

func commandDemo() {
	scheduler := newScheduler()

	house := smartHouse{}
	cLightsOn := cmdTurnLightsOn{&house}
	cLightsOff := cmdTurnLightsOff{&house}
	cMakeCoffee := cmdMakeCoffee{&house}

	scheduler.addTask(time.Now().Add(time.Second*5), cMakeCoffee)
	fmt.Println("Добавили задачу через 5 секунд сварить кофе")
	scheduler.addTask(time.Now().Add(time.Second*12), cLightsOff)
	fmt.Println("Добавили задачу через 12 секунд выключить свет")
	house.printState()
	scheduler.stop()
	fmt.Println("Вызываем команду 'включить свет'")
	cLightsOn.execute()
	house.printState()
	fmt.Println("Ждем 13 секунд...")
	time.Sleep(time.Second * 13)
	house.printState()
}
