package pattern

import (
	"fmt"
	"math"
	"math/rand"
	"time"
)

/*
	Реализовать паттерн «фасад».
Объяснить применимость паттерна, его плюсы и минусы,а также реальные примеры использования данного примера на практике.
	https://en.wikipedia.org/wiki/Facade_pattern
*/

// Фасад - структурный паттерн, который подаёт простой интерфейс к сложной системе.
// В приведенном примере рассматриваются сервисы геоинформиции и фасад,
// позволяющий получить расстояние между двумя адресами по кратчайшему пути, используя
// методы различных интерфейсов.

// интерфейсы геосервисов
type (
	// point - точка на карте
	point struct {
		lat, lon float64
	}

	// geoService - сервис карт. Нас интересует метод, преобразующий адрес в точку.
	geoService interface {
		addressToPoint(string) (point, error)
		// ...
		// другие методы сервиса
	}

	// route - итерируемый маршрут по дорогам, состоящий из нескольких точек.
	route interface {
		reset()
		next() bool
		getNode() point
	}

	// routeService - сервис работы с маршрутами.
	routeService interface {
		// getRoute - генератор машрута между данными точками.
		getRoute([]point) route
		// ...
		// другие методы сервиса
	}

	// routeCalc сервис, считающий длину маршрута.
	routeCalc interface {
		calculateDistance(route) (float64, error)
		// ...
		// другие методы сервиса
	}
)

// routeFacade - фасад для определения расстояния маршрута из пункта А в пункт Б.
// Использует три сервиса, предоставляя удобный интерфейс.
type routeFacade struct {
	gs geoService
	rs routeService
	rc routeCalc
}

// calculateDistance - метод фасада, считающий длину маршрута между двумя адресами.
func (f routeFacade) calculateDistance(addressA, addressB string) (float64, error) {
	pointA, err := f.gs.addressToPoint(addressA)
	if err != nil {
		return 0, err
	}
	pointB, err := f.gs.addressToPoint(addressB)
	if err != nil {
		return 0, err
	}
	route := f.rs.getRoute([]point{pointA, pointB})
	return f.rc.calculateDistance(route)
}

// mock-реализации интерфейсов геосервисов

type mockGeoService struct{}

// addresssToPoint назначает адресу случайную точку
func (gs mockGeoService) addressToPoint(address string) (point, error) {
	p := point{
		lat: 40 + rand.Float64()*20,
		lon: 20 + rand.Float64()*10,
	}
	fmt.Printf("mockGeoService: координаты точки %q - %v\n", address, p)
	return p, nil
}

type mockRoute struct {
	points   []point
	iterator int
}

func (r *mockRoute) reset() {
	r.iterator = 0
}
func (r *mockRoute) next() bool {
	if r.iterator >= len(r.points)-1 {
		return false
	}
	r.iterator++
	fmt.Printf("mockRoute: переходим к точке %v\n", r.points[r.iterator])
	return true
}
func (r *mockRoute) getNode() point {
	return r.points[r.iterator]
}

type mockRouteService struct{}

func (rs mockRouteService) getRoute(points []point) route {
	return &mockRoute{points: points, iterator: -1} // лучший маршрут - по прямой!
}

type mockRouteCalc struct{}

// calculateDistance довольно честно считает длину маршрута.
func (rc mockRouteCalc) calculateDistance(r route) (float64, error) {
	const (
		degreeLat = 111.0 // длина градуса широты в км.
		degreeLon = 111.3 // длина градуса долготы в км.
	)
	square := func(n float64) float64 { return n * n }
	distance := -1.0
	var previous point
	for r.next() {
		current := r.getNode()
		if distance == -1 {
			previous = current
			distance = 0
			continue
		}
		fmt.Printf("mockRouteCalc: считаем расстояние между точками %v и %v\n", previous, current)
		distance += math.Sqrt(square(degreeLat*(current.lat-previous.lat)) +
			square(degreeLon*(current.lon-previous.lon)))
		previous = current
	}
	return distance, nil
}

// клиентский код
func demoFacade() {
	rand.Seed(time.Now().UnixNano())
	facade := routeFacade{
		gs: &mockGeoService{},
		rs: &mockRouteService{},
		rc: &mockRouteCalc{},
	}
	d, err := facade.calculateDistance("Москва", "Санкт-Петербург")
	if err != nil {
		panic(err)
	}
	fmt.Printf("\nРасстояние между Москвой и Петербургом по нашим данным составляет %.2f км.\n", d)
}
