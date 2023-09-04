package api

import (
	"encoding/json"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
)

type MessageIn struct {
	ID string `json:"id"`
}

type Message struct {
	Zakupka  *Zakupka `json:"zakupka"`
	Message  string   `json:"message" example:""`
	Status   bool     `json:"status" example:"true"`
	LogLevel string   `json:"logLevel,omitempty"`
}

func GetContractCard(log *logrus.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var msg MessageIn
		w.Header().Set("Content-Type", "application/json")
		var zakupka = New(log)
		b, _ := ioutil.ReadAll(r.Body)
		if err := json.Unmarshal(b, &msg); err != nil {
			send, _ := json.Marshal(Message{Message: "не корректный json", Status: false})
			w.Write(send)
			return
		}
		zakupka.RequestEpz(msg.ID)
		if zakupka.Error != "" {
			send, _ := json.Marshal(Message{Message: zakupka.Error, Status: false, Zakupka: zakupka})
			w.Write(send)
			return
		}
		send, _ := json.Marshal(Message{Status: true, Zakupka: zakupka})
		w.Write(send)
		return
	})
}

func ChangeLogLevelHandler(l *logrus.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var msg Message
		b, _ := ioutil.ReadAll(r.Body)
		if err := json.Unmarshal(b, &msg); err != nil {
			send, _ := json.Marshal(Message{Message: "не корректный json", Status: false})
			w.Write(send)
			return
		}
		level, err := logrus.ParseLevel(msg.LogLevel)
		if err != nil {
			send, _ := json.Marshal(Message{Message: "не корректное значние уровня логирования: " + msg.LogLevel + " ошибка:" + err.Error(), Status: false})
			w.Write(send)
			return
		}
		l.Level = level
		send, _ := json.Marshal(Message{Status: true, Message: "уровень логирования изменен на: " + msg.LogLevel})
		w.Write(send)
		return
	})
}
