package api

import (
	"github.com/PuerkitoBio/goquery"
	"github.com/sirupsen/logrus"
	//"io/ioutil"
	//"fmt"
	//"bytes"
	"crypto/tls"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

type MyTime struct {
	time.Time
}

type Zakupka struct {
	Wg                         sync.WaitGroup    `json:"-"`
	Mutex                      sync.RWMutex      `json:"-"`
	ID                         string            `json:"id"`
	NumberContract             string            `json:"numberContract"`
	ReesterNumberContract      string            `json:"reesterNumberContract"`
	IdentificationCodePurchase string            `json:"identificationCodePurchase"`
	SubjectContract            string            `json:"subjectContract"`
	StateContract              string            `json:"stateContract"`
	DateContract               MyTime            `json:"dateContract,omitempty"`
	Stages                     []Stage           `json:"stages"`
	DateStartContract          MyTime            `json:"dateStartContract,omitempty"`
	PenaltyInfo                []PenaltyInfo     `json:"penaltyInfo"`
	Dopnik                     []string          `json:"dopnik"`
	SubWorkers                 []SubWorker       `json:"subWorkers"`
	SumContract                float64           `json:"sumContract"`
	SumByYear                  map[int64]float64 `json:"sumByYear"`
	DataEndContract            MyTime            `json:"dataEndContract,omitempty"`
	Customer                   Customer          `json:"customer,omitempty"`
	log                        *logrus.Logger    `json:"-"`
	Error                      string            `json:"error"`
}

type Customer struct {
	FullName         string `json:"fullname"`
	ShortName        string `json:"shortName"`
	RegistrationDate MyTime `json:"registrationDate"`
	Inn              string `json:"inn"`
	Kpp              string `json:"kpp"`
	Okpo             string `json:"okpo"`
}

type SubWorker struct {
	FullName            string `json:"fullname"`
	ShortName           string `json:"shortName"`
	Country             string `json:"country"`
	RegistrationAddress string `json:"registrationAddress"`
	PostalAddress       string `json:"postalAddress"`
	RegistrationDate    MyTime `json:"registrationDate"`
	Phone               string `json:"phone"`
	Email               string `json:"email"`
	Inn                 string `json:"inn"`
	Kpp                 string `json:"kpp"`
	Okpo                string `json:"okpo"`
}

type Stage struct {
	DateStart     MyTime     `json:"dateStart"`
	DateEnd       MyTime     `json:"dateEnd"`
	CostCompleted float64    `json:"costCompleted"`
	CostPaid      float64    `json:"costPaid"`
	Document      []Document `json:"document"`
	Penalty       string     `json:"penalty"`
	Completed     string     `json:"completed"`
	URL           *string    `json:"-"`
}

type Document struct {
	Name                 string  `json:"name"`
	DateSigning          MyTime  `json:"dateSigning"`
	CostCompleted        float64 `json:"costCompleted"`
	CostPaid             float64 `json:"costPaid"`
	NonExecutionContract string  `json:"nonExecutionContract"`
}

type PenaltyInfo struct {
	Payer   string  `json:"payer"`
	Reason  string  `json:"reason"`
	Demand  string  `json:"demand"`
	Accrued float64 `json:"accrued"`
	Paid    float64 `json:"paid"`
}

type Payment struct {
	Year int64
	Paid float64
}

type PaymentByYear struct {
	Payment map[int]Payment
}

func (t *MyTime) MarshalJSON() ([]byte, error) {
	if t.IsZero() {
		return []byte("null"), nil
	}
	return []byte(t.Format("\"" + time.RFC3339 + "\"")), nil
}

// UnmarshalJSON implements the json.Unmarshaler interface.
// The time is expected to be a quoted string in RFC 3339 format.
/*func (t *MyTime) UnmarshalJSON(data []byte) (err error) {

	// by convention, unmarshalers implement UnmarshalJSON([]byte("null")) as a no-op.
	if bytes.Equal(data, []byte("null")) {
		return nil
	}

	// Fractional seconds are handled implicitly by Parse.
	tt, err := time.Parse("\""+time.RFC3339+"\"", string(data))
	*t = MyTime{&tt}
	return
}*/

var replaceArray = []string{"«Дополнительное соглашение к контракту»", "  ", "\n", "№", "₽", " ", "<br>"}
var replacerPhone = []string{"--", ")", "(", "  "}
var client = &http.Client{
	Transport: tr,
	Timeout:   10 * time.Second,
}

var backoffSchedule = []time.Duration{
	1 * time.Second,
	3 * time.Second,
	10 * time.Second,
}

var tr = &http.Transport{
	TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
}

func New(l *logrus.Logger) *Zakupka {
	return &Zakupka{
		log:       l,
		SumByYear: make(map[int64]float64),
	}
}

// повтор запросов если не обработались с первого раза
func requestRepeat(req *http.Request, id string) (resp *http.Response, err error) {
	for _, backoff := range backoffSchedule {
		resp, err = client.Do(req)
		if err != nil {
			err = fmt.Errorf("запрос данных закупки по ID: %s : %s", id, err)
			time.Sleep(backoff)
			continue
		}
		// Read response
		//data_b, err := ioutil.ReadAll(resp.Body)
		//e.log.Info(string(data_b))
		if resp.StatusCode != 200 {
			err = fmt.Errorf("статус выполнения запроса: %d код ошибки %s", resp.StatusCode, resp.Status)
			time.Sleep(backoff)
			continue
		}
		break

	}
	return resp, err
}

func GetDocumentInfo(id string, inc int) ([]Document, error) {
	var documents []Document
	r := regexp.MustCompile("\\s+")
	url := "https://zakupki.gov.ru" + id + "&itemIndex=" + strconv.Itoa(inc*50) + "&pageSize=50"
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Content-Type", "text/html;charset=UTF-8")
	req.Header.Set("Content-Encoding", "gzip")
	resp, err := requestRepeat(req, id)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения HTML: %s", err.Error())
	}
	doc.Find("table").Each(func(i int, s *goquery.Selection) {
		s.Find("tr").Each(func(indextr int, rowtr *goquery.Selection) {
			var document Document
			var correct int
			check := rowtr.Find("td")
			if check.Length() == 5 {
				correct = 1
			}
			if check.Length() >= 5 {
				rowtr.Find("td").EachWithBreak(func(indextd int, rowtd *goquery.Selection) bool {

					indextd = indextd + correct
					switch indextd {
					case 0:
						return true
					case 1:
						document.Name = strings.Trim(strings.Replace(r.ReplaceAllString(rowtd.Text(), " "), "\n", " ", -1), " ")
					case 2:
						t, _ := convertTime(r.ReplaceAllString(rowtd.Text(), ""))
						document.DateSigning = MyTime{t}
					case 3:
						document.CostCompleted = SumToFloat((strings.Trim(replacer(replaceArray, rowtd.Text()), " ")))
					case 4:
						sumStr := strings.Split(rowtd.Text(), "(")
						document.CostPaid = SumToFloat(strings.Trim(strings.Replace(r.ReplaceAllString(sumStr[0], ""), " ", "", -1), " "))
					case 5:
						document.NonExecutionContract = strings.Trim(replacer(replaceArray, rowtd.Text()), " ")
						documents = append(documents, document)
						return false
					}
					return true
				})
			}
		})
	})
	return documents, nil
}

func (z *Zakupka) GetProcessInfo(id string) {
	defer z.Wg.Done()
	var stages []Stage
	var penaltyInfos []PenaltyInfo
	url := "https://zakupki.gov.ru/epz/contract/contractCard/process-info.html?reestrNumber=" + id
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Content-Type", "text/html;charset=UTF-8")
	req.Header.Set("Content-Encoding", "gzip")
	resp, err := requestRepeat(req, id)
	if err != nil {
		z.saveError(err)
		return
	}
	//data_b, err := ioutil.ReadAll(resp.Body)
	//z.log.Info(string(data_b))
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		z.saveError(fmt.Errorf("ошибка чтения HTML: %s", err.Error()))
		return
	}
	doc.Find("table.blockInfo__table").Each(func(i int, s *goquery.Selection) {
		s.Find("td.tableBlock__col").EachWithBreak(func(indextr int, rowhtml *goquery.Selection) bool {
			if (rowhtml.Index() >= 0) && len(rowhtml.Get(0).Attr) >= 0 {
				if strings.Contains(rowhtml.Get(0).Attr[0].Val, "rowExpand") {
					var stage Stage
					rowhtml.Find("span.general-chevron-handler").Each(func(index int, chevronhtml *goquery.Selection) {
						if typeDocument, ok := chevronhtml.Attr("data-content-id"); ok {
							if strings.Contains(typeDocument, "execution") {
								if url, ok := chevronhtml.Attr("data-url"); ok {
									stage.URL = &url
								}
							}
						}
					})
					b := rowhtml.Next()
					if (b.Index() >= 0) && (strings.Contains(b.Get(0).Attr[0].Val, "tableBlock__col_center")) {
						var flag bool
						stageSplit := strings.Split(replacer(replaceArray, rowhtml.Next().Text()), "-")
						if len(stageSplit) > 1 {
							flag = true
							ts, _ := convertTime(strings.Replace(stageSplit[0], " ", "", -1))
							stage.DateStart = MyTime{ts}
							te, _ := convertTime(strings.Replace(stageSplit[1], " ", "", -1))
							stage.DateEnd = MyTime{te}

						}
						stageSplitFrom := strings.Split(replacer(replaceArray, rowhtml.Next().Text()), "По ")
						if len(stageSplitFrom) > 1 && !flag {
							ts, _ := convertTime(strings.Replace(stageSplitFrom[1], " ", "", -1))
							stage.DateEnd = MyTime{ts}
						}
						stageSplitTo := strings.Split(replacer(replaceArray, rowhtml.Next().Text()), "От ")
						if len(stageSplitTo) > 1 && !flag {
							ts, _ := convertTime(strings.Replace(stageSplitTo[1], " ", "", -1))
							stage.DateStart = MyTime{ts}

						}
					}
					c := b.Next()
					d := c.Next()
					if (d.Index() >= 0) && (c.Index() >= 0) && (strings.Contains(c.Get(0).Attr[0].Val, "tableBlock__col_right")) {
						stage.CostCompleted = SumToFloat((strings.Trim(replacer(replaceArray, c.Text()), " ")))
						stage.CostPaid = SumToFloat((strings.Trim(replacer(replaceArray, d.Text()), " ")))
					}
					f := d.Next().Next()
					g := f.Next()
					if (f.Index() >= 0) && (strings.Contains(f.Get(0).Attr[0].Val, "tableBlock__col_first")) {
						stage.Penalty = strings.Trim(replacer(replaceArray, f.Text()), " ")
						stage.Completed = strings.Trim(replacer(replaceArray, g.Text()), " ")
					}
					stages = append(stages, stage)
				}
			}
			return true

		})
	})
	doc.Find("table.table").Each(func(i int, s *goquery.Selection) {
		s.Find("tr").Each(func(indextr int, rowtr *goquery.Selection) {
			var penaltyInfo PenaltyInfo
			rowtr.Find("td").EachWithBreak(func(indextd int, rowtd *goquery.Selection) bool {
				ftd := rowtd.Find("span.general-chevron-handler")
				if ftd.Length() == 0 && indextd == 0 {
					return false
				}
				switch indextd {
				case 0:
					return true
				case 1:
					penaltyInfo.Payer = strings.Trim(replacer(replaceArray, rowtd.Text()), " ")
				case 2:
					penaltyInfo.Reason = strings.Trim(replacer(replaceArray, rowtd.Text()), " ")
				case 3:
					penaltyInfo.Demand = strings.Trim(replacer(replaceArray, rowtd.Text()), " ")
				case 4:
					penaltyInfo.Accrued = SumToFloat((strings.Trim(replacer(replaceArray, rowtd.Text()), " ")))
				case 5:
					penaltyInfo.Paid = SumToFloat((strings.Trim(replacer(replaceArray, rowtd.Text()), " ")))
					penaltyInfos = append(penaltyInfos, penaltyInfo)
					return false
				}
				return true
			})
		})
	})
	if len(stages) > 0 {
		for key, value := range stages {
			for i := 0; i < 40; i++ {
				if value.URL != nil {
					document, err := GetDocumentInfo(*value.URL, i)
					if err != nil {
						z.log.Error(err)
						continue
					}
					//z.log.Warnf("url %s документ %s", *value.URL, document)
					stages[key].Document = append(stages[key].Document, document...)
					//fmt.Println("счетчик", i)
					//fmt.Println("колл-во", len(document))
					if len(document) < 50 {
						break
					}
				}
			}
		}
		z.Mutex.Lock()
		z.Stages = stages
		z.Mutex.Unlock()
	}
	if len(penaltyInfos) > 0 {
		z.Mutex.Lock()
		z.PenaltyInfo = penaltyInfos
		z.Mutex.Unlock()
	}
}

func (z *Zakupka) GetCommonInfo(id string) {
	var numberContract, dateContract, dataEndContract, fullName, shortName, inn, kpp, registrationDate, reesterNumberContract, identificationCodePurchase, dateStartContract, subjectContract, stateContract, okpo string
	var dopnik []string
	var subWorkers []SubWorker
	var indexCountry, indexRegistrationAddress, indexPostalAddress, indexPhoneEmail int = 10, 10, 10, 10
	defer z.Wg.Done()
	url := "https://zakupki.gov.ru/epz/contract/contractCard/common-info.html?reestrNumber=" + id
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Content-Type", "text/html;charset=UTF-8")
	req.Header.Set("Content-Encoding", "gzip")
	resp, err := requestRepeat(req, id)
	if err != nil {
		z.saveError(err)
		return
	}
	defer resp.Body.Close()
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		z.saveError(fmt.Errorf("ошибка чтения HTML: %s", err.Error()))
		return
	}
	doc.Find("div.cardMainInfo__section").Each(func(i int, s *goquery.Selection) {
		text := s.Find("span.cardMainInfo__title").Text()
		if text == "Контракт" {
			numberContract = "№" + strings.Trim(replacer(replaceArray, s.Find("span.cardMainInfo__content").Text()), " ")
		}
		if text == "Заключение контракта" {
			dateContract = s.Find("span.cardMainInfo__content").Text()
		}
	})
	doc.Find("section.blockInfo__section").Each(func(i int, s *goquery.Selection) {
		if strings.Contains(s.Find("span.section__title").Text(), "Реквизиты документа, являющегося основанием") {
			dopnik = append(z.Dopnik, "№"+replacer(replaceArray, s.Find("span.section__info").Text()))
		}
		if strings.Contains(s.Find("span.section__title").Text(), "Реестровый номер контракта") {
			reesterNumberContract = s.Find("span.section__info").Text()
		}
		if strings.Contains(s.Find("span.section__title").Text(), "Идентификационный код закупки (ИКЗ)") {
			identificationCodePurchase = s.Find("span.section__info").Text()
		}
		if strings.Contains(s.Find("span.section__title").Text(), "Полное наименование заказчика") {
			fullName = strings.Trim(replacer(replaceArray, s.Find("span.section__info").Text()), " ")
		}
		if strings.Contains(s.Find("span.section__title").Text(), "Сокращенное наименование заказчика") {
			shortName = s.Find("span.section__info").Text()
		}
		if strings.Contains(s.Find("span.section__title").Text(), "ИНН") {
			inn = strings.TrimSpace(s.Find("span.section__info").Text())
		}
		if strings.Contains(s.Find("span.section__title").Text(), "КПП") {
			kpp = strings.TrimSpace(s.Find("span.section__info").Text())
		}
		if strings.Contains(s.Find("span.section__title").Text(), "Код ОКПО") {
			okpo = s.Find("span.section__info").Text()
		}
		if strings.Contains(s.Find("span.section__title").Text(), "Дата постановки на учет в налоговом органе") {
			registrationDate = s.Find("span.section__info").Text()
		}
		if strings.Contains(s.Find("span.section__title").Text(), "Предмет контракта") && subjectContract == "" {
			subjectContract = s.Find("span.section__info").Text()
		}
		if strings.Contains(s.Find("span.section__title").Text(), "Статус контракта") {
			stateContract = strings.Trim(replacer(replaceArray, s.Find("span.section__info").Text()), " ")
		}
		if strings.Contains(s.Find("span.section__title").Text(), "Дата начала исполнения контракта") {
			splitDataStartContract := strings.Split(s.Find("span.section__info").Text(), " ")
			dateStartContract = strings.TrimSpace(splitDataStartContract[0])
		}
		if strings.Contains(s.Find("span.section__title").Text(), "Дата окончания исполнения контракта") {
			splitDataEndContract := strings.Split(s.Find("span.section__info").Text(), " ")
			dataEndContract = strings.TrimSpace(splitDataEndContract[0])
		}
	})
	doc.Find("table.blockInfo__table").Each(func(i int, tableHeadSubWorkers *goquery.Selection) {
		tableHeadSubWorkers.Find("th.tableBlock__col.tableBlock__col_header").Each(func(i int, thSubWorkers *goquery.Selection) {
			if strings.Contains(thSubWorkers.Text(), "Страна, код") {
				indexCountry = i
			}
			if strings.Contains(thSubWorkers.Text(), "Адрес места нахождения") {
				indexRegistrationAddress = i
			}
			if strings.Contains(thSubWorkers.Text(), "Почтовый адрес") {
				indexPostalAddress = i
			}
			if strings.Contains(thSubWorkers.Text(), "Телефон, электронная почта") {
				indexPhoneEmail = i
			}
		})
	})
	doc.Find("tbody.tableBlock__body").Each(func(i int, tableSubWorkers *goquery.Selection) {
		var subWorker SubWorker
		tableSubWorkers.Find("tr").Each(func(indextr int, trSubWorkers *goquery.Selection) {
			trSubWorkers.Find("td").Each(func(indextd int, tdSubWorkers *goquery.Selection) {
				if indextd == 0 {
					if !trSubWorkers.Is("br") {
						text, _ := tdSubWorkers.Html()
						strArrayWithBr := strings.Split(text, "\n")
						if len(strArrayWithBr) > 1 {
							for _, value := range strArrayWithBr {
								if strings.Contains(value, "<br/>") {
									textWithFormat := strings.TrimSpace(strings.Replace(value, "<br/>", " ", -1))
									subWorker.FullName, subWorker.ShortName = getFullNameAndShortName(textWithFormat)
									break
								}
							}
						}
					}
					if subWorker.FullnameEmpty() {
						strArray := strings.Split(tdSubWorkers.Text(), "\n")
						for _, value := range strArray {
							row := strings.ReplaceAll(value, "  ", "")
							if len(row) > 0 {
								subWorker.FullName, subWorker.ShortName = getFullNameAndShortName(row)
								break
							}
						}
					}
					var indexInn, indexKpp, indexOkpo, indexDateRegister int
					tdSubWorkers.Find("span").Each(func(indexspan int, spanSubWorkers *goquery.Selection) {
						if strings.Contains(spanSubWorkers.Text(), "ИНН") {
							indexInn = indexspan + 1
						}
						if strings.Contains(spanSubWorkers.Text(), "КПП") {
							indexKpp = indexspan + 1
						}
						if strings.Contains(spanSubWorkers.Text(), "Код по ОКПО") {
							indexOkpo = indexspan + 1
						}
						if strings.Contains(spanSubWorkers.Text(), "Дата") {
							indexDateRegister = indexspan + 1
						}
						if indexInn != 0 && indexInn == indexspan {
							subWorker.Inn = spanSubWorkers.Text()
						}
						if indexKpp != 0 && indexKpp == indexspan {
							subWorker.Kpp = spanSubWorkers.Text()
						}
						if indexOkpo != 0 && indexOkpo == indexspan {
							subWorker.Okpo = spanSubWorkers.Text()
						}
						if indexDateRegister != 0 && indexDateRegister == indexspan {
							src, _ := convertTime(spanSubWorkers.Text())
							subWorker.RegistrationDate = MyTime{src}
						}
					})
				}
				if indextd == indexCountry {
					arr := strings.Split(tdSubWorkers.Text(), "\n")
					for _, value := range arr {
						row := strings.ReplaceAll(value, "  ", "")
						if len(row) > 0 {
							subWorker.Country = strings.Trim(row, " ")
							break
						}
					}
				}
				if indextd == indexRegistrationAddress {
					subWorker.RegistrationAddress = strings.Trim(replacer(replaceArray, tdSubWorkers.Text()), " ")
				}
				if indextd == indexPostalAddress {
					subWorker.PostalAddress = strings.Trim(replacer(replaceArray, tdSubWorkers.Text()), " ")
				}
				if indextd == indexPhoneEmail {
					arr := strings.Split(tdSubWorkers.Text(), "\n")
					for _, value := range arr {
						row := strings.Trim(replacer(replacerPhone, value), "")
						if strings.Contains(row, ",") {
							phoneArray := strings.Split(row, ",")
							row = phoneArray[0]
						}
						if len(row) > 0 {
							if strings.Contains(row, "@") || strings.Contains(row, ".") {
								subWorker.Email = strings.Trim(row, " ")
							} else {
								row = strings.ReplaceAll(row, "-", "")
								if phone, ok := PhoneNumberToInternationFormat(strings.Trim(row, " ")); ok {
									subWorker.Phone = phone
								}
							}
						}
					}
				}

			})
		})
		subWorkers = append(subWorkers, subWorker)
	})
	sumContract := replacer(replaceArray, doc.Find("div.sectionMainInfo").Find("div.price").Find("span.cardMainInfo__content.cost").Text())
	z.Mutex.Lock()
	tc, _ := convertTime(dateContract)
	z.DateContract = MyTime{tc}
	tsc, _ := convertTime(dateStartContract)
	z.DateStartContract = MyTime{tsc}
	trc, _ := convertTime(registrationDate)
	z.Customer.RegistrationDate = MyTime{trc}
	z.SumContract = SumToFloat(sumContract)
	z.Customer.FullName = fullName
	z.Customer.ShortName = shortName
	z.Customer.Inn = inn
	z.Customer.Kpp = kpp
	z.Customer.Okpo = okpo
	z.NumberContract = numberContract
	z.SubjectContract = subjectContract
	z.Dopnik = dopnik
	z.StateContract = stateContract
	z.SubWorkers = subWorkers
	z.ReesterNumberContract = reesterNumberContract
	z.IdentificationCodePurchase = identificationCodePurchase
	tec, _ := convertTime(strings.TrimSpace(dataEndContract))
	z.DataEndContract = MyTime{tec}
	z.Mutex.Unlock()
	return
}

func (z *Zakupka) saveError(err error) {
	z.log.Error(err)
	z.Mutex.Lock()
	z.Error = err.Error()
	z.Mutex.Unlock()
}

func (z *Zakupka) GetPaymentInfo(id string) {
	reDigit := regexp.MustCompile("[0-9]+")
	reLetter := regexp.MustCompile("[а-яА-ЯёЁa-zA-Z]")
	var replaceArray = []string{"  ", "\n", "№", "₽", ",", "на", "год", " ", "Этап:"}
	var replaceSum = []string{"  ", "₽", " ", " ", "\n"}
	var sum = NewPayment()

	defer z.Wg.Done()
	url := "https://zakupki.gov.ru/epz/contract/contractCard/payment-info-and-target-of-order.html?reestrNumber=" + id
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Content-Type", "text/html;charset=UTF-8")
	req.Header.Set("Content-Encoding", "gzip")
	resp, err := requestRepeat(req, id)
	if err != nil {
		z.saveError(err)
		return
	}
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		z.saveError(fmt.Errorf("ошибка чтения HTML: %s", err.Error()))
		return
	}
	doc.Find("thead").EachWithBreak(func(indexthead int, rowtd *goquery.Selection) bool {
		rowtd.Find("th").Each(func(indexth int, tableheading *goquery.Selection) {
			if indexth != 0 {
				heading := strings.Trim(replacer(replaceArray, tableheading.Text()), " ")
				yearInt := YearToInt(getYearOnString(heading))
				sum.Payment[indexth-1] = Payment{Year: yearInt}
			}
		})
		return false
	})
	doc.Find("table.mt-3").Each(func(index int, tablehtml *goquery.Selection) {
		tablehtml.Find("tbody").Each(func(indextable int, table *goquery.Selection) {
			table.Find("tr").Each(func(indextr int, tabletr *goquery.Selection) {
				tabletr.Find("td").Each(func(indexth int, tablecell *goquery.Selection) {
					if indexth != 0 {
						row := strings.Trim(replacer(replaceSum, tablecell.Text()), " ")
						payment, ok := sum.Payment[indexth-1]
						if ok {
							containsDigit := reDigit.FindAllString(row, -1)
							if len(containsDigit) > 0 {
								containsLetter := reLetter.FindAllString(row, -1)
								if len(containsLetter) == 0 {
									payment.Paid = payment.Paid + SumToFloat(row)
									sum.Payment[indexth-1] = payment
								}
							}
						}
					}
				})
			})
		})
	})
	z.Mutex.Lock()
	z.SumByYear = sum.ToMap()
	z.Mutex.Unlock()
	return
}

func (z *Zakupka) RequestEpz(id string) {
	if id == "" {
		z.saveError(fmt.Errorf("пустой идентификационный номер госзакупки"))
		return
	}
	z.ID = id
	z.Wg.Add(3)
	go z.GetCommonInfo(id)
	go z.GetPaymentInfo(id)
	go z.GetProcessInfo(id)
	z.Wg.Wait()
}

func (p *PaymentByYear) ToMap() map[int64]float64 {
	var m = make(map[int64]float64)
	for _, value := range p.Payment {
		m[value.Year] = value.Paid
	}
	return m
}

func NewPayment() *PaymentByYear {
	return &PaymentByYear{
		Payment: make(map[int]Payment),
	}
}

func checkInArray(s string, a []string) bool {
	for _, value := range a {
		if s == value {
			return true
		}
	}
	return false
}

func convertTime(t string) (time.Time, error) {
	layout := "02.01.2006"
	c, err := time.Parse(layout, t)
	if err != nil {
		return c, err
	}
	return c, nil
}

func replacer(s []string, origin string) string {
	for _, value := range s {
		origin = strings.Replace(origin, value, "", -1)
	}
	return origin
}

func SumToFloat(origin string) float64 {
	if !(strings.Contains(origin, ",")) {
		return 0
	}
	origin = strings.Replace(origin, ",", ".", -1)
	s, _ := strconv.ParseFloat(origin, 64)
	return s
}

func YearToInt(origin string) int64 {
	i, _ := strconv.ParseInt(origin, 10, 64)
	return i
}

func PhoneNumberToInternationFormat(s string) (string, bool) {

	s = ParseNum(s) // выделяем цифры
	var number = []rune(s)
	if len(number) == 0 {
		return s, false
	}
	if len(number) == 10 {
		number = append([]rune("7"), number...)
		number = append([]rune("+"), number...)
	}
	if len(number) == 11 && number[0] == rune('8') {
		number[0] = rune('7')
		number = append([]rune("+"), number...)
	}
	if len(number) == 11 && number[0] != rune('+') && number[0] == rune('7') {
		number = append([]rune("+"), number...)
	}
	return string(number), true
}

func getYearOnString(s string) string {
	re := regexp.MustCompile("[0-9]+")
	arrayYear := re.FindAllString(s, -1)
	for _, value := range arrayYear {
		if len(value) == 4 {
			return value
		}
	}
	return s
}

func getFullNameAndShortName(s string) (fullName, shortName string) {
	if strings.Contains(s, "(") {
		nameArray := strings.Split(s, "(")
		fullName = nameArray[0]
		shortName = strings.Replace(nameArray[1], ")", "", -1)
		return
	}
	fullName = s
	return
}

func (s *SubWorker) FullnameEmpty() bool {
	if s.FullName == "" {
		return true
	}
	return false
}

func ParseNum(s string) (str string) {
	nLen := 0
	for i := 0; i < len(s); i++ {
		if b := s[i]; '0' <= b && b <= '9' {
			nLen++
		}
	}
	var n = make([]int, 0, nLen)
	for i := 0; i < len(s); i++ {
		if b := s[i]; '0' <= b && b <= '9' {
			n = append(n, int(b)-'0')
		}
	}
	for _, value := range n {
		str = str + strconv.Itoa(value)
	}
	return
}
