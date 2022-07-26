package main

import (
	"context"
	"encoding/gob"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

/*
=== HTTP server ===

Реализовать HTTP сервер для работы с календарем. В рамках задания необходимо работать строго со стандартной HTTP библиотекой.
В рамках задания необходимо:
	1. Реализовать вспомогательные функции для сериализации объектов доменной области в JSON.
	2. Реализовать вспомогательные функции для парсинга и валидации параметров методов /create_event и /update_event.
	3. Реализовать HTTP обработчики для каждого из методов API, используя вспомогательные функции и объекты доменной области.
	4. Реализовать middleware для логирования запросов
Методы API: POST /create_event POST /update_event POST /delete_event GET /events_for_day GET /events_for_week GET /events_for_month
Параметры передаются в виде www-url-form-encoded (т.е. обычные user_id=3&date=2019-09-09).
В GET методах параметры передаются через queryString, в POST через тело запроса.
В результате каждого запроса должен возвращаться JSON документ содержащий либо {"result": "..."} в случае успешного выполнения метода,
либо {"error": "..."} в случае ошибки бизнес-логики.

В рамках задачи необходимо:
	1. Реализовать все методы.
	2. Бизнес логика НЕ должна зависеть от кода HTTP сервера.
	3. В случае ошибки бизнес-логики сервер должен возвращать HTTP 503. В случае ошибки входных данных
	 (невалидный int например) сервер должен возвращать HTTP 400. В случае остальных ошибок сервер должен
	  возвращать HTTP 500. Web-сервер должен запускаться на порту указанном в конфиге и выводить в лог каждый обработанный запрос.
	4. Код должен проходить проверки go vet и golint.
*/

// Условием выполнения заданий является разместить весь код в одном файле.
// На практике целесообразно ползоваться пакетами Go как архитектурными слоями.

// CalendarAPI содержит обработчики htttp-запросов сервиса календаря.
type CalendarAPI struct {
	storage EventStorage
}

// NewCalendar создаёт новый объект CalendarAPI.
func NewCalendar(s EventStorage) *CalendarAPI {
	return &CalendarAPI{
		storage: s,
	}
}

// CreateEvent создает новое событие в календаре.
//
// POST /create_event
// параметры (* = обязательный):
//	- *user_id		ID пользователя (uuid)
//	- *date 		дата в формате dd.mm.yyyy
//	- time 			локальное время hh:mm
//	- place 		место
//	- description 	описание события
func (c CalendarAPI) CreateEvent(w http.ResponseWriter, r *http.Request) {
	const logHeader = "createEvent"
	// проверяем метод
	if r.Method != http.MethodPost {
		returnError(w, logHeader, "", http.StatusMethodNotAllowed)
		return
	}
	defer r.Body.Close()

	// извлекаем параметры из запроса
	queryUserID := r.FormValue("user_id")
	if queryUserID == "" {
		returnError(w, logHeader, "missing parameter: user_id", http.StatusBadRequest)
		return
	}
	userID, err := uuid.Parse(queryUserID)
	if err != nil {
		returnError(w, logHeader, fmt.Sprintf("incorrect user ID: %v", err), http.StatusBadRequest)
		return
	}
	queryDate := r.FormValue("date")
	if queryDate == "" {
		returnError(w, logHeader, "missing parameter: date", http.StatusBadRequest)
		return
	}
	// time, place и description - необязательные параметры
	queryTime := r.FormValue("time")
	when, err := parseWhen(queryDate, queryTime)
	if err != nil {
		returnError(w, logHeader, fmt.Sprintf("incorrect date or time: %v", err), http.StatusBadRequest)
		return
	}
	queryPlace := r.FormValue("place")
	queryDescription := r.FormValue("description")
	event := Event{
		ID:     uuid.New(),
		UserID: userID,
		When:   when,
		Where:  queryPlace,
		What:   queryDescription,
	}
	// вызываем метод EventStorage для сохранения события
	if err := c.storage.Add(event); err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, ErrEventAlreadyExists) {
			status = http.StatusBadRequest
		}
		returnError(w, logHeader, err.Error(), status)
		return
	}
	returnResult(w, "event successfully added", http.StatusCreated)
	log.Printf("%s: created event %+v", logHeader, event)
}

// UpdateEvent ищет в хранилище событие с указанным event_id (обязательный праметр),
// и обновляет его поля в соответствии с переданными в запросе.
//
// POST /update_event
// параметры (* = обязательный):
//	- *event_id		ID события
//	- user_id		ID пользователя (uuid)
//	- date 			дата в формате dd.mm.yyyy
//	- time 			локальное время hh:mm
//	- place 		место
//	- description 	описание события
func (c CalendarAPI) UpdateEvent(w http.ResponseWriter, r *http.Request) {
	const logHeader = "updateEvent"
	// проверяем метод
	if r.Method != http.MethodPost {
		returnError(w, logHeader, "", http.StatusMethodNotAllowed)
		return
	}
	defer r.Body.Close()

	// извлекаем параметры
	queryEventID := r.FormValue("event_id")
	if queryEventID == "" {
		returnError(w, logHeader, "missing parameter: event_id", http.StatusBadRequest)
		return
	}
	eventID, err := uuid.Parse(queryEventID)
	if err != nil {
		returnError(w, logHeader, fmt.Sprintf("incorrect event ID: %v", err), http.StatusBadRequest)
		return
	}
	// ищем событие с указанным ID
	event, err := c.storage.Get(eventID)
	if err != nil {
		// устанавливаем статус в зависимости от типа ошибки
		status := http.StatusInternalServerError
		if errors.Is(err, ErrEventNotFound) {
			status = http.StatusNotFound
		}
		returnError(w, logHeader, err.Error(), status)
		return
	}

	// если есть параметр - обновляем его

	queryUserID := r.FormValue("user_id")
	if queryUserID != "" {
		userID, err := uuid.Parse(queryUserID)
		if err != nil {
			returnError(w, logHeader, fmt.Sprintf("incorrect user ID: %v", err), http.StatusBadRequest)
			return
		}
		event.UserID = userID
	}

	// поскольку дата и время хранятся в одном поле типа time.Time,
	// пытаемся смержить с имеющимися датой и временем
	var dateStr, timeStr string
	queryDate := r.FormValue("date")
	queryDateOk := queryDate != ""
	if !queryDateOk {
		dateStr = event.When.Format("02.01.2006")
	} else {
		dateStr = queryDate
	}
	queryTime := r.FormValue("time")
	queryTimeOk := queryTime != ""
	if !queryTimeOk {
		timeStr = event.When.Format("15:04")
	} else {
		timeStr = queryTime
	}
	// если был передан хотя бы один параметр - обновляем поле, предварительно "срастив"
	// дату с временем.
	if queryDateOk || queryTimeOk {
		when, err := parseWhen(dateStr, timeStr)
		if err != nil {
			returnError(w, logHeader, fmt.Sprintf("incorrect date or time: %v", err), http.StatusBadRequest)
			return
		}
		event.When = when
	}
	queryPlace := r.FormValue("place")
	if queryPlace != "" {
		event.Where = queryPlace
	}
	queryDescription := r.FormValue("description")
	if queryDescription != "" {
		event.What = queryDescription
	}

	// вызываем метод EventStorage
	if err := c.storage.Update(event); err != nil {
		returnError(w, logHeader, err.Error(), http.StatusInternalServerError)
		return
	}
	returnResult(w, "event successfully updated", http.StatusOK)
	log.Printf("%s: updated event %+v", logHeader, event)
}

// DeleteEvent удаляет ищет событие с переданным ID и удаляет его.
//
// POST /delete_event
// параметр:
// *event_id
func (c CalendarAPI) DeleteEvent(w http.ResponseWriter, r *http.Request) {
	const logHeader = "deleteEvent"
	// проверяем метод (думаю, в это м случае правильнее было бы использовать http метод DELETE)
	if r.Method != http.MethodPost {
		returnError(w, logHeader, "", http.StatusMethodNotAllowed)
		return
	}
	defer r.Body.Close()

	// получаем ID события
	queryEventID := r.FormValue("event_id")
	if queryEventID == "" {
		returnError(w, logHeader, "missing parameter: event_id", http.StatusBadRequest)
		return
	}
	eventID, err := uuid.Parse(queryEventID)
	if err != nil {
		returnError(w, logHeader, fmt.Sprintf("incorrect event ID: %v", err), http.StatusBadRequest)
		return
	}

	// вызываем метод EventStorage
	if err := c.storage.Delete(eventID); err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, ErrEventNotFound) {
			status = http.StatusNotFound
		}
		returnError(w, logHeader, err.Error(), status)
		return
	}
	returnResult(w, fmt.Sprintf("event %v successfully deleted", eventID), http.StatusNoContent)
	log.Printf("%s: deleted event %v", logHeader, eventID)
}

// GetDayEvents - получить все события указанного дня.
//
// GET /events_for_day
// параметры:
// *user_id
// *date
func (c CalendarAPI) GetDayEvents(w http.ResponseWriter, r *http.Request) {
	const logHeader = "getDayEvents"
	// проверяем метод и получаем параметры
	userID, day, ok := getEventParams(w, r, logHeader)
	if !ok {
		return // ошибки уже обработаны
	}
	events, err := c.storage.GetByDay(userID, day)
	if err != nil {
		returnError(w, logHeader, err.Error(), http.StatusInternalServerError)
		return
	}
	returnEvents(w, logHeader, events)
}

// GetWeekEvents - получить все события за неделю, начиная с указанного дня.
// GET /events_for_week
// параметры:
// *user_id
// *date
func (c CalendarAPI) GetWeekEvents(w http.ResponseWriter, r *http.Request) {
	const logHeader = "getWeekEvents"

	// проверяем метод и получаем параметры
	userID, week, ok := getEventParams(w, r, logHeader)
	if !ok {
		return // ошибки уже обработаны
	}
	events, err := c.storage.GetForWeek(userID, week)
	if err != nil {
		returnError(w, logHeader, err.Error(), http.StatusInternalServerError)
		return
	}
	returnEvents(w, logHeader, events)
}

// GetMonthEvents получить все события за месяц, начиная с указанного дня.
//
// GET /events_for_month
// параметры:
// *user_id
// *date
func (c CalendarAPI) GetMonthEvents(w http.ResponseWriter, r *http.Request) {
	const logHeader = "getMonthEvents"
	userID, month, ok := getEventParams(w, r, logHeader)
	if !ok {
		return // ошибки уже обработаны
	}
	events, err := c.storage.GetForMonth(userID, month)
	if err != nil {
		returnError(w, logHeader, err.Error(), http.StatusInternalServerError)
		return
	}
	returnEvents(w, logHeader, events)
}

// getEventParams - проверка метода (должен быть GET) и извлечение из запроса параметров
// для обработчиков /events_for_day, /events_for_week, /events_for_month
// Функция обрабатывает и логирует возникшие ошибки.
// параметры:
//	- *user_id
//	- *date
func getEventParams(w http.ResponseWriter, r *http.Request, logHeader string) (uuid.UUID, time.Time, bool) {
	if r.Method != http.MethodGet {
		returnError(w, logHeader, "", http.StatusMethodNotAllowed)
		return uuid.Nil, time.Time{}, false
	}
	userIDstr := r.FormValue("user_id")
	if userIDstr == "" {
		returnError(w, logHeader, "missing parameter: user_id", http.StatusBadRequest)
		return uuid.Nil, time.Time{}, false
	}
	userID, err := uuid.Parse(userIDstr)
	if err != nil {
		returnError(w, logHeader, fmt.Sprintf("incorrect user ID: %v", err), http.StatusBadRequest)
		return uuid.Nil, time.Time{}, false
	}
	dateStr := r.FormValue("date")
	if dateStr == "" {
		returnError(w, logHeader, "missing parameter: date", http.StatusBadRequest)
		return uuid.Nil, time.Time{}, false
	}
	t, err := time.Parse("02.01.2006", dateStr)
	if err != nil {
		returnError(w, logHeader, fmt.Sprintf("incorrect date format: %s", dateStr), http.StatusBadRequest)
		return uuid.Nil, time.Time{}, false
	}
	return userID, t, true
}

// parseWhen конвертирует дату и время (последнее - при наличии) в переменную типа time.Time.
// формат даты: "02.01.2006"
// формат времени "15:04"
func parseWhen(dateStr, timeStr string) (time.Time, error) {
	var layout, str string
	if len(timeStr) > 0 {
		// если есть время, присовокупляем его
		layout = "02.01.2006 15:04"
		str = fmt.Sprintf("%s %s", dateStr, timeStr)
	} else {
		layout = "02.01.2006"
		str = dateStr
	}
	result, err := time.Parse(layout, str)
	if err != nil {
		return time.Time{}, err
	}
	return result, nil
}

// returnResult устанавливает требуемый статус-код в заголовке ответа
// и записывает в тело ответа JSON со строкой результата.
func returnResult(w http.ResponseWriter, result string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	fmt.Fprintf(w, `{"result": %s}`, result)
}

// returnError логирует возникшую ошибку и записывает её в тело ответа,
// установив требуемый статус-код в заголовке.
func returnError(w http.ResponseWriter, logHeader, err string, status int) {
	log.Printf("%s: %s", logHeader, err)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	if err == "" {
		fmt.Fprintf(w, `{"error": "%s"}`, http.StatusText(status))
		return
	}
	fmt.Fprintf(w, `{"error": "%s: %s"}`, http.StatusText(status), err)
}

// returnEvents устанавливает статус 200 OK и записывает в тело ответа
// JSON с массивом найденных событий (может быть пустым).
func returnEvents(w http.ResponseWriter, logHeader string, events []Event) {
	type result struct {
		Result []Event `json:"result"`
	}
	w.Header().Set("Content-Type", "application/json")
	res := result{Result: events}
	if err := json.NewEncoder(w).Encode(res); err != nil {
		returnError(w, logHeader, err.Error(), http.StatusInternalServerError)
		return
	}
}

// LoggerMiddleware логирует информацию о всех http-запросах.
func LoggerMiddleware(next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer next.ServeHTTP(w, r)
		sb := strings.Builder{}
		sb.WriteString(fmt.Sprintf("Request %s %s from %q\n", r.Method, r.RequestURI, r.RemoteAddr))
		sb.WriteString("Header:\n")
		for k, v := range r.Header {
			sb.WriteString(fmt.Sprintf("\t%s: %q\n", k, v))
		}
		sb.WriteString("FormValues:\n")
		if err := r.ParseForm(); err != nil {
			sb.WriteString(fmt.Sprintf("could not parse form: %s", err))
			log.Println("\t", sb.String())
			return
		}
		for k, v := range r.Form {
			sb.WriteString(fmt.Sprintf("%s: %q\n", k, v))
		}
		log.Println(sb.String())
		return
	})
}

// Event - событие в календаре.
type Event struct {
	// UUID для идентификаторов выбраны для того, чтобы из сервиса
	// не "торчали" ключи.

	// ID - уникальный идентификатор события.
	ID uuid.UUID
	// UserID - ID пользователя
	UserID uuid.UUID

	When  time.Time
	Where string
	What  string
}

// EventStorage - интерфейс хранилища событий в календаре
type EventStorage interface {
	// Add добавляет событие в хранилище. Если в хранилище уже имеется
	// событие с ID равным переданному - возвращается ошибка ErrEventAlreadyExists.
	Add(Event) error
	// Update перезаписывает событие с переданным ID, если оно есть в хранилище.
	// В случае отсутствия возвращается ErrEventNotFound.
	Update(Event) error
	// Delete удаляет из хранилища событие с данным ID.
	// В случае отсутствия возвращается ErrEventNotFound.
	Delete(uuid.UUID) error
	// Get возвращает событие с данным ID.
	// В случае отсутствия возвращается ErrEventNotFound.
	Get(uuid.UUID) (Event, error)
	// GetByDay возвращает все события пользователя с данным userID за сутки от
	// переданного момента. В случае отсутствия событий возвращается пустой массив.
	GetByDay(userID uuid.UUID, t time.Time) ([]Event, error)
	// GetForWeek возвращает все события пользователя с данным userID за неделю от
	// переданного момента. В случае отсутствия событий возвращается пустой массив.
	GetForWeek(userID uuid.UUID, t time.Time) ([]Event, error)
	// GetForMonth возвращает все события пользователя с данным userID за месяц от
	// переданного момента. В случае отсутствия событий возвращается пустой массив.
	GetForMonth(userID uuid.UUID, t time.Time) ([]Event, error)
}

// Ошибки EventStorage.
var (
	ErrEventAlreadyExists = errors.New("event already exists")
	ErrEventNotFound      = errors.New("event not found")
)

var _ EventStorage = (*InmemEventStorage)(nil)

// InmemEventStorage - имплементация EventStorage.
// Хранилище расположено в оперативной памяти. Периодически все данные
// сохраняются в файле (при наличии изменений).
type InmemEventStorage struct {
	mu *sync.RWMutex
	// repo является хранилищем событий
	repo map[uuid.UUID]Event
	// modified устанавливается, когда данные в хранилище обновляются и
	// их необходимо сохранить на диск.
	modified bool
	// stopCh - канал, закрытие которого останавливает repoSaver.
	stopCh chan struct{}
	wg     *sync.WaitGroup
}

const (
	// имя файла, в котором сохраняется repo.
	persistentStorageFile = "event_storage.gob"
	// интервал, с периодичностью которого происходят попытки
	// сохранения данных.
	storageFlushInterval = time.Second * 5
)

// NewInmemEventStorage создаёт новое хранилище и запускает воркер, сохраняющий
// изменения в файл.
func NewInmemEventStorage() (*InmemEventStorage, error) {
	s := &InmemEventStorage{
		mu:     &sync.RWMutex{},
		repo:   make(map[uuid.UUID]Event),
		stopCh: make(chan struct{}, 1),
		wg:     &sync.WaitGroup{},
	}

	if err := s.readStorageFile(); err != nil {
		log.Printf("inmemEventStorage: could not open file %s: %s", persistentStorageFile, err)
	} else {
		log.Printf("inmemEventStorage: %d entrie(s) successfully read from the file", len(s.repo))
	}
	s.wg.Add(1)
	go s.repoSaver()

	return s, nil
}

// repoSaver - воркер, сохраняющий данные хранилища в файл с заданной периодичностью.
func (s *InmemEventStorage) repoSaver() {
	log.Println("inmemEventStorage: persistent repository saver started")
	flushTick := time.NewTicker(storageFlushInterval)
	for {
		select {
		case <-flushTick.C:
			s.saveRepo()
		case <-s.stopCh:
			// заканчиваем работу
			s.stopCh = nil
			flushTick.Stop()
			s.saveRepo() // сохраняем данные
			log.Println("inmemEventStorage: repoSaver stopped")
			s.wg.Done()
			return
		}
	}
}

// saveRepo сохраняет (при наличии изменений) содержимое хранилища в gob-файл.
func (s *InmemEventStorage) saveRepo() {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.modified {
		return
	}
	f, err := os.Create(persistentStorageFile)
	if err != nil {
		log.Printf("inmemEventStorage: repoSaver: ERROR: could not save data to the file: %v", err)
		return
	}
	defer f.Close()
	if err := gob.NewEncoder(f).Encode(s.repo); err != nil {
		log.Printf("inmemEventStorage: repoSaver: ERROR: could not save data to the file: %v", err)
		return
	}
	s.modified = false
	log.Println("inmemEventStorage: repoSaver: data successfully saved to the file")
}

// Close закрывает хранилище и останавливает воркер repoSaver.
func (s *InmemEventStorage) Close() {
	if s.stopCh == nil {
		return
	}
	close(s.stopCh)
	s.wg.Wait()
	s.repo = nil
	log.Println("inmemEventStorage closed")
}

// readStorageFile читает содержимое хранилища из gob-файла.
func (s *InmemEventStorage) readStorageFile() error {
	f, err := os.Open(persistentStorageFile)
	if err != nil {
		return err
	}
	if err := gob.NewDecoder(f).Decode(&s.repo); err != nil {
		return err
	}
	s.modified = false
	return nil
}

// Add реализует интерфейс EventStorage.
func (s *InmemEventStorage) Add(e Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.repo[e.ID]; ok {
		return ErrEventAlreadyExists
	}
	s.repo[e.ID] = e
	s.modified = true
	return nil
}

// Update реализует интерфейс EventStorage.
func (s *InmemEventStorage) Update(e Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.repo[e.ID]; !ok {
		return ErrEventNotFound
	}
	s.repo[e.ID] = e
	s.modified = true
	return nil
}

// Delete реализует интерфейс EventStorage.
func (s *InmemEventStorage) Delete(eventID uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.repo[eventID]; !ok {
		return ErrEventNotFound
	}
	delete(s.repo, eventID)
	s.modified = true
	return nil
}

// Get реализует интерфейс EventStorage.
func (s *InmemEventStorage) Get(eventID uuid.UUID) (Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	event, ok := s.repo[eventID]
	if !ok {
		return Event{}, ErrEventNotFound
	}
	return event, nil
}

// GetByDay реализует интерфейс EventStorage.
func (s *InmemEventStorage) GetByDay(userID uuid.UUID, t time.Time) ([]Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]Event, 0)
	for _, event := range s.repo {
		if event.UserID == userID &&
			event.When.Sub(t) >= 0 &&
			t.AddDate(0, 0, 1).Sub(event.When) >= 0 {
			result = append(result, event)
		}
	}
	return result, nil
}

// GetForWeek реализует интерфейс EventStorage.
func (s *InmemEventStorage) GetForWeek(userID uuid.UUID, t time.Time) ([]Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]Event, 0)
	for _, event := range s.repo {
		if event.UserID == userID &&
			event.When.Sub(t) >= 0 &&
			t.AddDate(0, 0, 7).Sub(event.When) >= 0 {
			result = append(result, event)
		}
	}
	return result, nil
}

// GetForMonth реализует интерфейс EventStorage.
func (s *InmemEventStorage) GetForMonth(userID uuid.UUID, t time.Time) ([]Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]Event, 0)
	for _, event := range s.repo {
		if event.UserID == userID &&
			event.When.Sub(t) >= 0 &&
			t.AddDate(0, 1, 0).Sub(event.When) >= 0 {
			result = append(result, event)
		}
	}
	return result, nil
}

func main() {
	port := flag.String("p", "8080", "port")
	flag.Parse()
	// запускаем storage
	storage, err := NewInmemEventStorage()
	if err != nil {
		log.Fatal(err)
	}
	defer storage.Close()

	// устанавливаем роутер и прописываем маршруты
	api := NewCalendar(storage)
	router := http.NewServeMux()
	router.Handle("/create_event", LoggerMiddleware(api.CreateEvent))
	router.Handle("/update_event", LoggerMiddleware(api.UpdateEvent))
	router.Handle("/delete_event", LoggerMiddleware(api.DeleteEvent))
	router.Handle("/events_for_day", LoggerMiddleware(api.GetDayEvents))
	router.Handle("/events_for_week", LoggerMiddleware(api.GetWeekEvents))
	router.Handle("/events_for_month", LoggerMiddleware(api.GetMonthEvents))

	// устанавливаем http-сервер
	server := http.Server{
		Addr:    ":" + *port,
		Handler: router,
	}
	go func() {
		if err := server.ListenAndServe(); err != nil {
			log.Println(err)
		}
	}()
	log.Printf("Listening at %s...", server.Addr)

	// подписываемся на сигнал завершения и ждём
	sigTerm := make(chan os.Signal, 1)
	signal.Notify(sigTerm, os.Interrupt, os.Kill)
	<-sigTerm
	server.Shutdown(context.Background())
}
