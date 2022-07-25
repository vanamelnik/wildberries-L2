package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var user0ID = uuid.New().String()
var user1ID = uuid.New().String()

type respBody struct {
	Result []Event
}

// end-to end test
func TestCalendar(t *testing.T) {
	//запускаем сервис
	go main() // не знаю, так вообще делается?
	t.Run("Create", tCreate)
	t.Run("Update", tGetAndUpdate)
	t.Run("Get", tGet)
	t.Run("Delete", tDelete)
	os.Remove(persistentStorageFile)
}

func tCreate(t *testing.T) {
	tt := []struct {
		user        int
		date        string
		timeStr     string
		place       string
		description string
		wantErr     bool
	}{
		{
			user:        0,
			date:        "03.01.2022",
			timeStr:     "12:23",
			place:       "Клуб 2х2",
			description: "Торжественное мероприятие",
			wantErr:     false,
		},
		{
			user:        0,
			date:        "07.01.2022",
			timeStr:     "01:23",
			place:       "Клуб 3х3",
			description: "Неторжественное мероприятие",
			wantErr:     false,
		},
		{
			user:        0,
			date:        "31.12.2021",
			timeStr:     "23:55",
			place:       "Клуб 0х0",
			description: "Международный день тестировщика",
			wantErr:     false,
		},
		{
			user:        0,
			date:        "03.03.2022",
			timeStr:     "",
			place:       "Клуб 9х9",
			description: "Закрытие клуба",
			wantErr:     false,
		},
		{
			user:        0,
			date:        "43.01.2022",
			timeStr:     "",
			place:       "",
			description: "",
			wantErr:     true,
		},
		{
			user:        0,
			date:        "",
			timeStr:     "",
			place:       "",
			description: "",
			wantErr:     true,
		},
		{
			user:        0,
			date:        "13.01.2022",
			timeStr:     "12.00", // неверный формат
			place:       "",
			description: "",
			wantErr:     true,
		},
		{
			user:        1,
			date:        "07.11.1917",
			timeStr:     "05:00",
			place:       "Зимний дворец",
			description: "Захват Зимнего",
			wantErr:     false,
		},
	}
	for i, tc := range tt {
		t.Run(fmt.Sprintf("#%d", i), func(t *testing.T) {
			form := url.Values{}
			if tc.user == 0 {
				form["user_id"] = []string{user0ID}
			} else {
				form["user_id"] = []string{user1ID}
			}
			if tc.date != "" {
				form["date"] = []string{tc.date}
			}
			if tc.timeStr != "" {
				form["time"] = []string{tc.timeStr}
			}
			if tc.place != "" {
				form["place"] = []string{tc.place}
			}
			if tc.description != "" {
				form["description"] = []string{tc.description}
			}
			resp, err := http.PostForm("http://localhost:8080/create_event", form)
			assert.NoError(t, err)
			if tc.wantErr {
				assert.Equal(t, resp.StatusCode, http.StatusBadRequest)
				return
			}
			assert.Equal(t, resp.StatusCode, http.StatusCreated)
		})
	}
}

func tGetAndUpdate(t *testing.T) {
	uri := fmt.Sprintf("http://localhost:8080/events_for_day?user_id=%s&date=07.11.1917", user1ID)
	res := respBody{}
	resp, err := http.Get(uri)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, resp.Header["Content-Type"], "application/json")
	err = json.NewDecoder(resp.Body).Decode(&res)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, 1, len(res.Result))
	e := res.Result[0]
	assert.Equal(t, "Зимний дворец", e.Where)
	assert.Equal(t, "Захват Зимнего", e.What)
	form := url.Values{
		"user_id":  {user1ID},
		"event_id": {e.ID.String()},
		"time":     {"18:30"},
	}
	resp, err = http.PostForm("http://localhost:8080/update_event", form)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	resp, err = http.Get(uri)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, resp.Header["Content-Type"], "application/json")

	err = json.NewDecoder(resp.Body).Decode(&res)
	require.NoError(t, err)
	assert.Equal(t, 1, len(res.Result))
	e = res.Result[0]
	assert.Equal(t, "Зимний дворец", e.Where)
	assert.Equal(t, "Захват Зимнего", e.What)
	timeWant, _ := time.Parse("02.01.2006 15:04", "07.11.1917 18:30")
	assert.Equal(t, timeWant, e.When)
}

func tGet(t *testing.T) {
	tt := []struct {
		path    string
		date    string
		wantLen int
	}{
		{
			path:    "events_for_week",
			date:    "31.12.2021",
			wantLen: 2,
		},
		{
			path:    "events_for_month",
			date:    "31.12.2021",
			wantLen: 3,
		},
		{
			path:    "events_for_month",
			date:    "31.12.2024",
			wantLen: 0,
		},
	}
	for i, tc := range tt {
		t.Run(fmt.Sprintf("#%d %s", i, tc.path), func(t *testing.T) {
			uri := fmt.Sprintf("http://localhost:8080/%s?user_id=%s&date=%s",
				tc.path, user0ID, tc.date)
			res := respBody{}
			resp, err := http.Get(uri)
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode)
			assert.Contains(t, resp.Header["Content-Type"], "application/json")
			err = json.NewDecoder(resp.Body).Decode(&res)
			assert.NoError(t, err)
			assert.Equal(t, tc.wantLen, len(res.Result))
		})
	}
}

func tDelete(t *testing.T) {
	res := respBody{}
	uri := fmt.Sprintf("http://localhost:8080/events_for_day?user_id=%s&date=03.03.2022", user0ID)
	resp, err := http.Get(uri)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, resp.Header["Content-Type"], "application/json")
	err = json.NewDecoder(resp.Body).Decode(&res)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, 1, len(res.Result))

	form := url.Values{
		"event_id": {res.Result[0].ID.String()},
	}
	resp, err = http.PostForm("http://localhost:8080/delete_event", form)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	assert.Contains(t, resp.Header["Content-Type"], "application/json")

	resp, err = http.Get(uri)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, resp.Header["Content-Type"], "application/json")
	err = json.NewDecoder(resp.Body).Decode(&res)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, 0, len(res.Result))
}
