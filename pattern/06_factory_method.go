package pattern

import (
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
)

/*
	Реализовать паттерн «фабричный метод».
Объяснить применимость паттерна, его плюсы и минусы, а также реальные примеры использования данного примера на практике.
	https://en.wikipedia.org/wiki/Factory_method_pattern
*/

// В языке Go отсутствуют классы и объекты, и в полной мере паттерн "Фабричный метод" не может быть реализован.
// Однако интерфейсы Go позволяют описывать сущности, подобные Product, а их имплементация - ConcreteProduct.
// Применение данного шаблона позволяет отделить структуру от реализации.
// В нижеприведенном примере объявляется интерфейс хранилища заказов и две его реализации - in-memory и mock.

type Order struct {
	ID        uuid.UUID
	CreatedAt time.Time
	Items     []uuid.UUID
}

type (
	// OrderStorage - интерфейс, описывающий хранилище заказов (Order).
	OrderStorage interface {
		Get(id uuid.UUID) (*Order, error)
		Store(o Order) error
		Delete(id uuid.UUID) error
		Update(o Order) error
	}
	OrderStorageType int
)

var ErrNotFound = errors.New("order not found")

var _ OrderStorage = (*InmemOrderStorage)(nil)

// InmemOrderStorage - реализация интерфейса OrderStorage.
type InmemOrderStorage struct {
	mu         *sync.RWMutex
	repository map[uuid.UUID]Order
}

// NewInmemOrderStorage - констркутор. Можно считать фабричным методом.
func NewInmemOrderStorage() InmemOrderStorage {
	return InmemOrderStorage{
		mu:         &sync.RWMutex{},
		repository: make(map[uuid.UUID]Order),
	}
}

func (s *InmemOrderStorage) Get(id uuid.UUID) (*Order, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	o, ok := s.repository[id]
	if !ok {
		return nil, ErrNotFound
	}
	return &o, nil
}

func (s *InmemOrderStorage) Store(o Order) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.repository[o.ID]; ok {
		return errors.New("order already exists")
	}
	s.repository[o.ID] = o
	return nil
}

func (s *InmemOrderStorage) Update(o Order) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.repository[o.ID]; !ok {
		return ErrNotFound
	}
	s.repository[o.ID] = o
	return nil
}

func (s *InmemOrderStorage) Delete(id uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.repository[id]; !ok {
		return ErrNotFound
	}
	delete(s.repository, id)
	return nil
}

var _ OrderStorage = (*MockOrderStorage)(nil)

// MockOrderStorage - мок реализация хранилища.
type MockOrderStorage struct{}

func (m MockOrderStorage) Get(id uuid.UUID) (*Order, error) {
	return nil, ErrNotFound
}
func (m MockOrderStorage) Store(o Order) error {
	return nil
}
func (m MockOrderStorage) Delete(id uuid.UUID) error {
	return nil
}
func (m MockOrderStorage) Update(o Order) error {
	return ErrNotFound
}
