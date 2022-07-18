package pattern

import (
	"fmt"
	"strings"
	"time"
)

/*
	Реализовать паттерн «посетитель».
Объяснить применимость паттерна, его плюсы и минусы, а также реальные примеры использования данного примера на практике.
	https://en.wikipedia.org/wiki/Visitor_pattern
*/

// Поведенческий патерн "Посетитель" используется, когда неким объектам нужно добавить функциональность,
// при этом не внося существенных изменений в код объекта.
// В этом примере мы рассматриваем боевой космический корабль и двух посетителей -
// damageChecker и repairer, которые "проходят" по всем узлам корабля, оценивают повреждения и ремонтируют.

// интерфейс посетителя
type visitor interface {
	visitEngine(*engine)
	visitShell(*shell)
	visitWeaponsBay(*weaponsBay)
}

// типы, которым добавляется функциональность

// двигатель космического корабля
type engine struct {
	name   string
	damage int
}

func (e *engine) accept(v visitor) {
	v.visitEngine(e)
}

// обшивка
type shell struct {
	name   string
	damage int
}

func (s *shell) accept(v visitor) {
	v.visitShell(s)
}

// оружейный отсек
type weaponsBay struct {
	name   string
	damage int
}

func (wb *weaponsBay) accept(v visitor) {
	v.visitWeaponsBay(wb)
}

// космический корабль
type battleSpaceShip struct {
	engines         []*engine
	frontShell      shell
	rearShell       shell
	rightWeaponsBay weaponsBay
	leftWeaponsBay  weaponsBay
}

func (s *battleSpaceShip) accept(v visitor) {
	for _, e := range s.engines {
		e.accept(v)
	}
	s.frontShell.accept(v)
	s.rearShell.accept(v)
	s.rightWeaponsBay.accept(v)
	s.leftWeaponsBay.accept(v)
}

// damageChecker - посетитель, проверяющий уровень разрушений в отсеках космического корабля
type damageChecker struct {
	damageMap map[string]int
}

// конструктор
func newChecker() damageChecker {
	return damageChecker{damageMap: make(map[string]int)}
}

// методы новой функциональности
func (c *damageChecker) visitEngine(e *engine) {
	fmt.Printf("Checking %s...\n", e.name)
	c.damageMap[e.name] = e.damage
}
func (c *damageChecker) visitShell(s *shell) {
	fmt.Printf("Checking %s...\n", s.name)
	c.damageMap[s.name] = s.damage
}
func (c *damageChecker) visitWeaponsBay(wb *weaponsBay) {
	fmt.Printf("Checking %s...\n", wb.name)
	c.damageMap[wb.name] = wb.damage
}

// damageReport - отчет о разрушениях в отсеках корабля
func (c *damageChecker) damageReport() string {
	sb := strings.Builder{}
	sb.WriteString("\nChecker damage report:\n------------------\n")
	var total, totalFull int
	for name, damage := range c.damageMap {
		total += damage
		totalFull += 100
		sb.WriteString(fmt.Sprintf("%s:\t%d%%\n", name, damage))
	}
	sb.WriteString(fmt.Sprintf("----------------\nTotal damage: %d%%\n", total*100/totalFull))
	return sb.String()
}

// repairer - посетитель-ремонтник. Чинит всё!
type repairer struct{}

func (r *repairer) visitEngine(e *engine) {
	if e.damage == 0 {
		fmt.Printf("%s is not damaged\n", e.name)
	}
	fmt.Printf("Repairing %s...", e.name)
	time.Sleep(time.Millisecond * 20 * time.Duration(e.damage))
	e.damage = 0
	fmt.Println(" OK")
}
func (r *repairer) visitShell(s *shell) {
	if s.damage == 0 {
		fmt.Printf("%s is not damaged\n", s.name)
	}
	fmt.Printf("Repairing %s...", s.name)
	time.Sleep(time.Millisecond * 20 * time.Duration(s.damage))
	s.damage = 0
	fmt.Println(" OK")
}
func (r *repairer) visitWeaponsBay(wb *weaponsBay) {
	if wb.damage == 0 {
		fmt.Printf("%s is not damaged\n", wb.name)
	}
	fmt.Printf("Repairing %s...", wb.name)
	time.Sleep(time.Millisecond * 20 * time.Duration(wb.damage))
	wb.damage = 0
	fmt.Println(" OK")
}

// клиентский код
func visitorDemo() {
	// создаём космический корабль
	ship := battleSpaceShip{
		engines: []*engine{
			{name: "Engine 1", damage: 8},
			{name: "Engine 2", damage: 27},
			{name: "Engine 3", damage: 0},
			{name: "Engine 4", damage: 52},
		},
		frontShell:      shell{name: "Front shell", damage: 87},
		rearShell:       shell{name: "Rear shell", damage: 67},
		rightWeaponsBay: weaponsBay{name: "Right weapons bay", damage: 3},
		leftWeaponsBay:  weaponsBay{name: "Left weapons bay", damage: 94},
	}
	// создаём посетителей
	checker := newChecker()
	mario := repairer{}

	// проверяющий проверяет...
	ship.accept(&checker)
	fmt.Println(checker.damageReport())

	// ремонтник чинит...
	ship.accept(&mario)

	// проверяем ещё раз, убеждаемся, что всё ОК
	ship.accept(&checker)
	fmt.Println(checker.damageReport())
}
