package main

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/url"
	"time"

	"GOzakupki/api"
	"GOzakupki/config"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/sirupsen/logrus"
)

var (
	conf = config.New()
	log  = logrus.New()
)

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
			transport := &http.Transport{
				Proxy:           http.ProxyURL(proxyURL),
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			}
			clientReg.Transport = transport
		}
	}

	if conf.S3.Endpoint != "" && conf.S3.Bucket != "" {
		minioOpts := &minio.Options{
			Creds:  credentials.NewStaticV4(conf.S3.AccessKey, conf.S3.SecretKey, ""),
			Region: conf.S3.Region,
			Secure: conf.S3.UseSSL,
		}
		mc, err := minio.New(conf.S3.Endpoint, minioOpts)
		if err != nil {
			log.Errorf("ошибка подключения к S3: %v", err)
		} else {
			exists, err := mc.BucketExists(context.Background(), conf.S3.Bucket)
			if err != nil {
				log.Errorf("ошибка проверки bucket %s: %v", conf.S3.Bucket, err)
			}
			if !exists {
				err = mc.MakeBucket(context.Background(), conf.S3.Bucket, minio.MakeBucketOptions{Region: conf.S3.Region})
				if err != nil {
					log.Errorf("ошибка создания bucket %s: %v", conf.S3.Bucket, err)
				} else {
					log.Infof("bucket %s создан", conf.S3.Bucket)
				}
			} else {
				log.Infof("bucket %s существует", conf.S3.Bucket)
			}
		}
	}

	log.Info("версия 0.6.7")
}

func main() {
	router := mux.NewRouter()
	router.Handle("/api/get", api.GetContractCard(log, conf)).Methods("POST", "OPTIONS")
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
