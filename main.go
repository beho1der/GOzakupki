package main

import (
	"GOzakupki/api"
	"GOzakupki/config"
	"crypto/tls"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"net/http"
	"net/url"
	"time"
)

var conf = config.New()
var log = logrus.New()

func init() {

	log.SetFormatter(&logrus.TextFormatter{ // настройки логирования
		FullTimestamp: true,
	})
	log.Level = conf.LogLevel
	clientReg := &http.Client{ // создаем новый http клиент
		Timeout: conf.HttpTimeWait * time.Second,
	}
	if conf.Main.Proxy { // добавляем прокси при наличии
		proxyURL, err := url.Parse(conf.Main.ProxyUrl)
		if err != nil {
			log.Errorf("не корректный адрес proxy:", err)
		} else {
			transport := &http.Transport{Proxy: http.ProxyURL(proxyURL),
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
			clientReg.Transport = transport
		}
	}

	log.Info("версия 0.5.1")

}

func main() {
	router := mux.NewRouter()
	router.Handle("/api/get", api.GetContractCard(log)).Methods("POST", "OPTIONS")
	router.Handle("/api/loglevel", api.ChangeLogLevelHandler(log)).Methods("POST", "OPTIONS")

	gzipHandler := handlers.CompressHandler(router)
	headersOk := handlers.AllowedHeaders([]string{"X-Requested-With", "Content-Type", "Authorization"})
	originsOk := handlers.AllowedOrigins([]string{"*"})
	methodsOk := handlers.AllowedMethods([]string{"GET", "HEAD", "POST", "PUT", "OPTIONS"})
	log.Info("HTTP сервер запущен на порту:", conf.Port)
	err := http.ListenAndServe(":"+conf.Port, handlers.CORS(originsOk, headersOk, methodsOk)(gzipHandler))
	if err != nil {
		log.Fatal("ошибка старта сервера:", err)
	}
}
