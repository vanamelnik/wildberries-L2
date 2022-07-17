package pattern

import (
	"errors"
	"fmt"
	"sync/atomic"
	"time"
)

/*
	Реализовать паттерн «состояние».
Объяснить применимость паттерна, его плюсы и минусы, а также реальные примеры использования данного примера на практике.
	https://en.wikipedia.org/wiki/State_pattern
*/

// Паттерн "состояние" применяется для создания объекта, обладающего набором дискретных состояний,
// и ведущего себя по-разному, в зависимости от текущего состояния.
// В приведеном примере мы используем модель домашнего робота, способного мыть посуду, мыть пол и мыть окна.
// Робот имеет два состояния - "работа" и "зарядка". В зависимости от уровня заряда батареи может быть
// осуществлено переключение состояния.

// интерфейс состояния робота
type robotState interface {
	cleanFloor(chargeLevel *int32) error
	washDishes(chargeLevel *int32) error
	washWindows(chargeLevel *int32) error
	goRecharge()
}

// ошибка, возникающая, когда уровень заряда низкий для выполнения работы
var chargeNeeded = errors.New("Charge level is low! - Go recharge!")

// тип состояния
type stType int

const (
	stTypeWork = iota
	stTypeCharge
)

// реализация состояния работы
type stateWork struct{}

func (s stateWork) cleanFloor(chargeLevel *int32) error {
	fmt.Println("Current battery level is ", *chargeLevel)
	if *chargeLevel < 10 {
		return chargeNeeded
	}
	fmt.Println("Cleaning the floor...")
	time.Sleep(time.Millisecond * 1000)
	return nil
}
func (s stateWork) washDishes(chargeLevel *int32) error {
	fmt.Println("Current battery level is ", *chargeLevel)
	if *chargeLevel < 10 {
		return chargeNeeded
	}
	fmt.Println("Washing the dishes...")
	time.Sleep(time.Millisecond * 1000)
	return nil
}
func (s stateWork) washWindows(chargeLevel *int32) error {
	fmt.Println("Current battery level is ", *chargeLevel)
	if *chargeLevel < 10 {
		return chargeNeeded
	}
	fmt.Println("Washing the windows...")
	time.Sleep(time.Millisecond * 1000)
	return nil
}
func (s stateWork) goRecharge() {
	fmt.Println("Going to charge...")
}

// реализация состояния зарядки
type stateCharge struct{}

func (s stateCharge) cleanFloor(chargeLevel *int32) error {
	fmt.Println("Could not clean floor - I'm recharging!")
	return nil
}
func (s stateCharge) washDishes(chargeLevel *int32) error {
	fmt.Println("Could not wash dishes - I'm recharging!")
	return nil
}
func (s stateCharge) washWindows(chargeLevel *int32) error {
	fmt.Println("Could not wash windows - I'm recharging!")
	return nil
}
func (s stateCharge) goRecharge() {
	fmt.Println("I'm already recharging!")
}

// объект "робот"
type robot struct {
	chargeLevel      int32
	currentStateType stType
	currentState     robotState
}

// конструктор, создающий нового роота и запускающего цикл управления батареей
func newRobot() *robot {
	r := robot{
		chargeLevel:      100,
		currentStateType: stTypeWork,
		currentState:     stateWork{},
	}
	// воркер, меняющий уровень текущего заряда батареи робота
	// в зависимости от текущего состояния
	go func() {
		fmt.Println("Robot started!")
		for {
			if r.chargeLevel%10 == 0 {
				fmt.Printf("\tbattery: %d%%\n", r.chargeLevel)
			}
			time.Sleep(time.Millisecond * 20)
			if r.currentStateType == stTypeCharge {
				if r.chargeLevel == 100 {
					fmt.Println("\tCharge complete!")
					r.changeState(stTypeWork)
					continue
				}
				atomic.AddInt32(&r.chargeLevel, 1)
				continue
			}
			if r.chargeLevel > 0 {
				atomic.AddInt32(&r.chargeLevel, -1)
				continue
			}
			fmt.Println("Battery is 0% - go to recharge!")
			r.goRecharge()
			r.changeState(stTypeCharge)
		}
	}()
	return &r
}

// команды роботу
func (r *robot) cleanFloor() {
	err := r.currentState.cleanFloor(&r.chargeLevel)
	if errors.Is(err, chargeNeeded) {
		fmt.Println(err)
		r.currentState.goRecharge()
		r.changeState(stTypeCharge)
	}
}
func (r *robot) washDishes() {
	err := r.currentState.washDishes(&r.chargeLevel)
	if errors.Is(err, chargeNeeded) {
		fmt.Println(err)
		r.currentState.goRecharge()
		r.changeState(stTypeCharge)
	}
}
func (r *robot) washWindows() {
	err := r.currentState.washWindows(&r.chargeLevel)
	if errors.Is(err, chargeNeeded) {
		fmt.Println(err)
		r.currentState.goRecharge()
		r.changeState(stTypeCharge)
	}
}
func (r *robot) goRecharge() {
	r.currentState.goRecharge()
}

// команда изменить состояние
func (r *robot) changeState(state stType) {
	switch state {
	case stTypeWork:
		r.currentState = stateWork{}
		r.currentStateType = stTypeWork
	case stTypeCharge:
		r.currentState = stateCharge{}
		r.currentStateType = stTypeCharge
	default:
		panic("wrong state type")
	}
}

func stateDemo() {
	r := newRobot()
	r.cleanFloor()
	r.washDishes()
	r.washWindows()              // здесь заряда уже мало
	r.changeState(stTypeCharge)  // меняем состояние извне
	r.washWindows()              // в состоянии зарядки робот не выполняет работу
	r.goRecharge()               // о робот и так заряжается, о чём он и сообщит.
	time.Sleep(2 * time.Second)  // ждем, пока зарядка окончится
	r.washWindows()              // теперь можно помыть окна
	time.Sleep(time.Second * 10) // далее смотрим, как батарея разряжается,потом снова заряжается и т.д.
}
