package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
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
	3. В случае ошибки бизнес-логики сервер должен возвращать HTTP 503. В случае ошибки входных данных (невалидный int например) сервер должен возвращать HTTP 400. В случае остальных ошибок сервер должен возвращать HTTP 500. Web-сервер должен запускаться на порту указанном в конфиге и выводить в лог каждый обработанный запрос.
	4. Код должен проходить проверки go vet и golint.
*/

type (
	CalendarAPI struct {
		storage EventStorage
	}

	Event struct {
		ID     uuid.UUID
		UserID uuid.UUID
		When   time.Time
		Where  string
		What   string
	}
)

func NewCalendar(s EventStorage) *CalendarAPI {
	return &CalendarAPI{
		storage: s,
	}
}

// POST /create_event
// параметры (* = обязательный):
//	- *user_id		ID пользователя (uuid)
//	- *date 		дата в формате dd.mm.yyyy
//	- time 			локальное время hh:mm
//	- place 		место
//	- description 	описание события
func (c CalendarAPI) CreateEvent(w http.ResponseWriter, r *http.Request) {
	logHeader := "createEvent"
	if r.Method != http.MethodPost {
		returnError(w, logHeader, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		returnError(w, logHeader, fmt.Sprintf("error when reading request body: %s", err.Error()), http.StatusBadRequest)
		return
	}
	values, err := url.ParseQuery(string(body))
	if err != nil {
		returnError(w, logHeader, fmt.Sprintf("error when reading request body: %s", err.Error()), http.StatusBadRequest)
		return
	}
	queryUserId, ok := values["user_id"]
	if !ok {
		returnError(w, logHeader, "missing parameter: user_id", http.StatusBadRequest)
		return
	}
	userID, err := uuid.Parse(queryUserId[0])
	if err != nil {
		returnError(w, logHeader, fmt.Sprintf("incorrect user ID: %v", err), http.StatusBadRequest)
		return
	}
	queryDate, ok := values["date"]
	if !ok {
		returnError(w, logHeader, "missing parameter: date", http.StatusBadRequest)
		return
	}
	timeStr := ""
	queryTime, ok := values["time"]
	if ok {
		timeStr = queryTime[0]
	}
	when, err := parseWhen(queryDate[0], timeStr)
	if err != nil {
		returnError(w, logHeader, fmt.Sprintf("incorrect date or time: %v", err), http.StatusBadRequest)
		return
	}
	queryPlace := values["place"]
	queryDescription := values["description"]
	if err := c.storage.Add(&Event{
		ID:     uuid.New(),
		UserID: userID,
		When:   when,
		Where:  queryPlace[0],
		What:   queryDescription[0],
	}); err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, ErrEventAlreadyExists) {
			status = http.StatusBadRequest
		}
		returnError(w, logHeader, err.Error(), status)
		return
	}
	returnResult(w, "event successfully added", http.StatusCreated)
}

// POST /update_event
// параметры (* = обязательный):
//	- *event_id		ID события
//	- user_id		ID пользователя (uuid)
//	- date 			дата в формате dd.mm.yyyy
//	- time 			локальное время hh:mm
//	- place 		место
//	- description 	описание события
func (c CalendarAPI) UpdateEvent(w http.ResponseWriter, r *http.Request) {
	logHeader := "updateEvent"
	if r.Method != http.MethodPost {
		returnError(w, logHeader, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		returnError(w, logHeader, fmt.Sprintf("error when reading request body: %s", err.Error()), http.StatusBadRequest)
		return
	}
	values, err := url.ParseQuery(string(body))
	if err != nil {
		returnError(w, logHeader, fmt.Sprintf("error when reading request body: %s", err.Error()), http.StatusBadRequest)
		return
	}
	queryEventId, ok := values["event_id"]
	if !ok {
		returnError(w, logHeader, "missing parameter: event_id", http.StatusBadRequest)
		return
	}
	eventID, err := uuid.Parse(queryEventId[0])
	if err != nil {
		returnError(w, logHeader, fmt.Sprintf("incorrect event ID: %v", err), http.StatusBadRequest)
		return
	}
	event, err := c.storage.Get(eventID)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, ErrEventNotFound) {
			status = http.StatusNotFound
		}
		returnError(w, logHeader, err.Error(), status)
		return
	}

	queryUserId, ok := values["user_id"]
	if ok {
		userID, err := uuid.Parse(queryUserId[0])
		if err != nil {
			returnError(w, logHeader, fmt.Sprintf("incorrect user ID: %v", err), http.StatusBadRequest)
			return
		}
		event.UserID = userID
	}

	var dateStr, timeStr string
	queryDate, dateOk := values["date"]
	if !dateOk {
		dateStr = event.When.Format("01.02.2006")
	} else {
		dateStr = queryDate[0]
	}
	queryTime, timeOk := values["time"]
	if !timeOk {
		timeStr = event.When.Format("15:04")
	} else {
		timeStr = queryTime[0]
	}
	if dateOk || timeOk {
		when, err := parseWhen(dateStr, timeStr)
		if err != nil {
			returnError(w, logHeader, fmt.Sprintf("incorrect date or time: %v", err), http.StatusBadRequest)
			return
		}
		event.When = when
	}
	queryPlace, ok := values["place"]
	if ok {
		event.Where = queryPlace[0]
	}
	queryDescription, ok := values["description"]
	if ok {
		event.What = queryDescription[0]
	}
	if err := c.storage.Update(event); err != nil {
		returnError(w, logHeader, err.Error(), http.StatusInternalServerError)
		return
	}
	returnResult(w, "event successfully added", http.StatusCreated)
}

// POST /delete_event
// GET /events_for_day
// GET /events_for_week
// GET /events_for_month

func parseWhen(dateStr, timeStr string) (time.Time, error) {
	var layout, str string
	if len(timeStr) > 0 {
		layout = "01.02.2006 15.04"
		str = fmt.Sprintf("%s %s", dateStr, timeStr)
	} else {
		layout = "02.03.2004"
		str = dateStr
	}
	result, err := time.Parse(layout, str)
	if err != nil {
		return time.Time{}, err
	}
	return result, nil
}

func returnResult(w http.ResponseWriter, result string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	fmt.Fprintf(w, `{"result": %s}`, result)
}

func returnError(w http.ResponseWriter, logHeader, err string, status int) {
	log.Printf("%s: %s", logHeader, err)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	fmt.Fprintf(w, `{"error": "%s: %s"}`, http.StatusText(status), err)
}

type EventStorage interface {
	Add(*Event) error
	Update(*Event) error
	Delete(uuid.UUID)
	Get(uuid.UUID) (*Event, error)
	GetByDay(userID uuid.UUID, t time.Time) (*Event, error)
	GetForWeek(userID uuid.UUID, t time.Time) (*Event, error)
	GetForMonth(userID uuid.UUID, t time.Time) (*Event, error)
}

var (
	ErrEventAlreadyExists = errors.New("event already exists")
	ErrEventNotFound      = errors.New("event not found")
)

func main() {

}
