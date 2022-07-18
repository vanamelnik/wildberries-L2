package pattern

import "fmt"

/*
	Реализовать паттерн «строитель».
Объяснить применимость паттерна, его плюсы и минусы, а также реальные примеры использования данного примера на практике.
	https://en.wikipedia.org/wiki/Builder_pattern
*/

// Порождающий паттерн "Строитель" используется для пошагового создания объектов с большим количеством опциональных параметров,
// для которых использование единого сложного конструктора нерационально.
// В языке Go применение паттерна нецелесообразно из-за необходимости создавать дополнительные объекты строителя и директора.
// Вместо этого повсеместно используется паттерн "функциональные опции".
//
// В примере описывается конструктор объекта "дом".

// сложный объект "дом"
type house struct {
	wallsMaterial string
	doorMaterial  string
	numFloors     uint
	numDoors      uint
	hasElevator   bool
}

func (h house) String() string {
	withElevator := ""
	if h.hasElevator {
		withElevator = ", with an elevator"
	}
	return fmt.Sprintf("%d-floors house, walls material - %s, doors material - %s, number of doors - %d%s.",
		h.numFloors, h.wallsMaterial, h.doorMaterial, h.numDoors, withElevator)
}

// интерфейс строителя
type builder interface {
	setMaterial()
	setNumFloors()
	setNumDoors()
	setElevator()
	getHouse() house
}

// конкретный строитель деревянных домов
type woodenHouseBuilder struct {
	h house
}

func (w *woodenHouseBuilder) setMaterial() {
	w.h.doorMaterial = "wood"
	w.h.wallsMaterial = "wood"
}

func (w *woodenHouseBuilder) setNumFloors() {
	w.h.numFloors = 2
}
func (w *woodenHouseBuilder) setNumDoors() {
	w.h.numDoors = 2
}
func (w *woodenHouseBuilder) setElevator() {
	w.h.hasElevator = false
}
func (w *woodenHouseBuilder) getHouse() house {
	return w.h
}

// конкретный строитель кирпичных домов
type brickHouseBuilder struct {
	h house
}

func (b *brickHouseBuilder) setMaterial() {
	b.h.doorMaterial = "wood"
	b.h.wallsMaterial = "bricks"
}

func (b *brickHouseBuilder) setNumFloors() {
	b.h.numFloors = 4
}
func (b *brickHouseBuilder) setNumDoors() {
	b.h.numDoors = 3
}
func (b *brickHouseBuilder) setElevator() {
	b.h.hasElevator = true
}
func (b *brickHouseBuilder) getHouse() house {
	return b.h
}

// генератор строителя по названию
func getBuilder(builderType string) builder {
	switch builderType {
	case "woodenHouse":
		return new(woodenHouseBuilder)
	case "brickHouse":
		return new(brickHouseBuilder)
	}
	return nil
}

// директор - знает последовательность действий и может руководить любым строителем
type director struct {
	b builder
}

func newDirector(builderType string) *director {
	d := &director{}
	d.setBuilder(builderType)
	return d
}

// задать строителя - одного директора можно использовать для разных строек
func (d *director) setBuilder(buiderType string) {
	d.b = getBuilder(buiderType)
}

// результат совместной работы директора и строителя - построенный дом
func (d director) buildHouse() house {
	d.b.setMaterial()
	d.b.setNumFloors()
	d.b.setNumDoors()
	d.b.setElevator()
	return d.b.getHouse()
}

func builderDemo() {
	// создаем директора с кирпичным строителем
	director := newDirector("brickHouse")
	brickHouse := director.buildHouse()
	fmt.Println(brickHouse.String())

	// теперь построим деревянный дом - меняем строителя
	director.setBuilder("woodenHouse")
	woodenHouse := director.buildHouse()
	fmt.Println(woodenHouse.String())
}
