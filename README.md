Сервис используется для взаимодействия с системой гос закупок(https://zakupki.gov.ru/) для получния данных о государственных закупках по идентификатору закупки. Предусмотренно повторение запросов при неудаче,а также парелельный запрос при вытаскивании всех эапов закупки.

Сборка: go get && go build  main.go  

					**Взаимодействие с API**
					
Взаимодействие возможно осуществлять по REST JSON API(POST), а также через RabbitMQ   

**REST JSON API:**      

Комманды:  
1. Получения данных о закупки по номеру из реестра:   
Запрос: POST http://server/api/get    
Тело запроса: {"id": "2920456270218000088"}  
*curl -X POST --data '{"id": "2920456270218000088","timeout":20,"repeat":2}'  http://127.0.0.1:8025/api/get*  
----- расширенный запрос -----
*curl -X POST --data '{"id": "2772838158719000703","timeout":20,"repeat":2}'  http://127.0.0.1:8025/api/get*

*id* - уникальный id госзакупки  
*timeout* - время ожидания ответа от госзакупок  
*repeat* - колл-во повторов запросов в госзакупки в случаи неудачного опроса(часто бывают проблемы с недоступностью)  
в итоге общее время ожидания ответа: waitTime = timeout + (repeat * timeout)


**Ответ**:
 ```json
{
    "zakupka": {
        "id": "2920456270218000088",
        "numberContract": "№202",
        "identificationCodePurchase": "192772838158777280100108280010000414",
        "dateContract": "2018-12-28T00:00:00Z",
        "dopnik": [
            "№4"
        ],
        "subWorkers": [
       {
        "fullname": "ОБЩЕСТВО С ОГРАНИЧЕННОЙ ОТВЕТСТВЕННОСТЬЮ \"СИБИРСКАЯ АПТЕКА\" (ОБЩЕСТВО С ОГРАНИЧЕННОЙ ОТВЕТСТВЕННОСТЬЮ \"СИБИРСКАЯ АПТЕКА\")",
        "country": "Российская Федерация",
        "registrationAddress": "630059, ОБЛАСТЬ НОВОСИБИРСКАЯ 54, Г. НОВОСИБИРСК, УЛ. ЦЕНТРАЛЬНАЯ, Д. 121",
        "postalAddress": "630059, ,; ОБЛ НОВОСИБИРСКАЯ, Г НОВОСИБИРСК;УЛ ЦЕНТРАЛЬНАЯ, ДОМ 121",
        "phone": "8-383-3478892",
        "email": "sib.apteka@mail.ru",
        "inn": 5409000810,
        "kpp": 540901001
       }
      ],
        "sumContract": 213106864,
        "sumByYear": {
            "2018": 195000000,
            "2019": 11375590,
            "2020": 6731270
        },
        "dataEndContract": "2020-06-30T00:00:00Z"
       "customer": {
        "fullname": "КРАЕВОЕ ГОСУДАРСТВЕННОЕ БЮДЖЕТНОЕ УЧРЕЖДЕНИЕ ЗДРАВООХРАНЕНИЯ \"ГОРОДСКАЯ БОЛЬНИЦА  2, Г. БИЙСК\"",
        "shortName": "КГБУЗ \"ГОРОДСКАЯ БОЛЬНИЦА № 2, Г. БИЙСК\"",
        "registrationDate": null,
        "inn": 2227021070,
        "kpp": 220401001,
        "okpo": 42339149
    },
    },
    "message": "",
    "status": true
}
```
*id* - уникальный id госзакупки    
*numberContract* - номер контракта  
*reesterNumberContract*   - реестровый номер контракта  
*identificationCodePurchase*   - Идентификационный код закупки (ИКЗ)    
*subjectContract*   - предмет контракта    
*stateContract*   - статус контракта  
*dateContract* - дата контракта на закупку  
*dateStartContract* - дата начала действия контракта   
*penaltyInfo* - массив обьектов содержаший штрафы за неисполнения контракта   
  *payer* -  наименования плательщика    
  *reason* - причина неустойки  
  *demand* -  документ с требованием  
  *accrued* - начисленно  
  *paid* - оплаченно  
*stages* - массив обьектов стадий исполнения контракта  
  *dateStart* -  дата начала этапа      
  *dateEnd* -  дата конца этапа   
  *costCompleted* - стойиость исполненных обязательств     
  *costPaid* - фактически оплаченно        
  *document* - массив документов этапа  
    *name* -  название документа  
    *dateSigning* -  дата подписание документа  
    *costCompleted* - стойиость исполненных обязательств    
    *costPaid* -  фактически оплаченно    
    *nonExecutionContract* -  описание не надлежащего исполнения                
    *penalty* -  неустойки(да\нет)  
    *completed* -  completed(да\нет)  
*dopnik* -  дополнительные соглашения к основному договору    
*subWorkers* - подрядчики     
  *fullname* - название организации  
  *country*  - страна  
  *registrationAddress* - адресс регистрации 
  *postalAddress* - почтовый код  
  *phone* - телефон
  *email* - электронный адрес
  *inn*   - ИНН
  *kpp*   - КПП
*customer* - заказчик     
  *fullname* - название организации  
  *shortName*  - короткое наименование    
  *registrationAddress* - адресс регистрации 
  *registrationDate* - дата регистрации  
  *inn*   - ИНН  
  *kpp*   - КПП  
  *okpo*  - ОКПО
*sumContract* - основная сумма контракта       
*sumByYear* - сумма в разрезе годов   
*dataEndContract* - дата конца контракта           
*validityContract* - дата окончания действия договора        
*message* - информация об ошибке        
*status* - false - если в процессе выполнения произошла ошибка, true - если ошибок нету   

2. Изменения уровня логирования:   
Запрос: POST http://server/api/loglevel body запроса пустое.  
Тело запроса: {"logLevel": "info"}
*возможные значения: error,info,warning,debug*  

**Ответ**:
 ```json
{
    "message": "уровень логирования изменен на: info",
    "status": true
}
```
*message* - информация выполненом действии         
*status* - false - если в процессе выполнения произошла ошибка, true - если ошибок нету   

**RebbitMQ API:**           
1. Получения данных о закупки по номеру из реестра:   
Запрос: RabbitMQ JSON  
Тело запроса: {"id": "2920456270218000088"}
 ```json
{
   "uuid":"a08f163d-1d46-4ed7-a7b1-50b97a1cc635",
   "service":{
      "name":"Gozakupki"
   },
   "action":"getData",
   "data":{
      
   },
   "needResponse":true,
   "type":"request",
   "system":{
      "queue":"amq.gen-FstyHBgIhmHLmr6Mnkf2Dw",
      "name":"fcp"
   }
}
```
**Ответ**:
 ```json
{
  "uuid": "a08f163d-1d46-4ed7-a7b1-50b97a1cc635",
  "action": "updateData",
  "entity": {
    "name": "",
    "query": {
      "isActive": {
        "$eq": false
      }
    },
    "linksMode": "",
    "attributes": null
  },
  "service": {
    "name": "Gozakupki"
  },
  "data": {
    "id": "2920456270218000088",
        "numberContract": "№202",

```

                **Переменные окружения**  
```
 ------ App --------
 APP_NAME  "gozakupki" - Имя приложения.    

 ------- Main ----------
PROXY_ENABLE "false" - Использовать ли прокси для всех внешних http запросов.  
PROXY_URL "http://127.0.0.1:3128" - IP или URL с портом для доступа к серверу.     

 ------ System----------
 LOG_LEVEL "info" - Уровень логирования(info/debug/warning/panic).  
 PORT "8005" - Порт по которому доступен http endpoint.  