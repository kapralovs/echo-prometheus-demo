package main

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo-contrib/prometheus"
	"github.com/labstack/echo/v4"
	prom "github.com/prometheus/client_golang/prometheus"
)

const cKeyMetrics = "custom_metrics" //ключ для хранимого в контексте запроса значения

type (
	User struct {
		ID      int    `json:"id,omitempty"`
		Name    string `json:"name,omitempty"`
		Age     int    `json:"age,omitempty"`
		IsAdult bool   `json:"is_adult,omitempty"`
	}

	Note struct {
		ID    int    `json:"id,omitempty"`
		Title string `json:"title,omitempty"`
		Text  string `json:"text,omitempty"`
	}
	Metrics struct { //Структура, хранящая набор метрик
		idConvCnt    *prometheus.Metric // метрика (счетчик)
		idConvErrCnt *prometheus.Metric // метрика (счетчик)
		customDur    *prometheus.Metric
	}
)

var (
	users = []*User{
		{ID: 1, Name: "Sam", Age: 15},
		{ID: 2, Name: "John", Age: 22, IsAdult: true},
		{ID: 3, Name: "Henrik", Age: 39, IsAdult: true},
	}
	notes = []*Note{
		{ID: 1, Title: "Homework", Text: "Math"},
		{ID: 2, Title: "Game info", Text: "Developer: Arkane Studios"},
		{ID: 3, Title: "Friend's email", Text: "friendsemail@example.com"},
	}
)

// Функция - конструктор для создания и инициализации структуры Metrics
func NewMetrics() *Metrics {
	return &Metrics{
		idConvCnt: &prometheus.Metric{
			Name:        "conversions_count",
			Description: "id URL param conversions count",
			Type:        "counter_vec",
			Args:        []string{"conv_type", "entity", "result"},
		},
		idConvErrCnt: &prometheus.Metric{
			Name:        "conversions_err_count",
			Description: "id URL param conversions err count",
			Type:        "counter_vec",
			Args:        []string{"conv_type", "entity", "result"},
		},
		customDur: &prometheus.Metric{
			Name:        "custom_duration_seconds",
			Description: "Custom duration observations.",
			Type:        "histogram_vec",
			Args:        []string{"label_one", "label_two"},
			Buckets:     prom.DefBuckets, // or your Buckets
		},
	}
}

// Миддлварь, которая сохраняет структуру Metrics в контекст запроса
func (m *Metrics) AddCustomMetricsMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		c.Set(cKeyMetrics, m)
		return next(c)
	}
}

// Увеличение счетчика и установление
func (m *Metrics) IncConversionCount(labelOne, labelTwo, labelThree string) {
	labels := prom.Labels{"conv_type": labelOne, "entity": labelTwo, "result": labelThree}
	m.idConvCnt.MetricCollector.(*prom.CounterVec).With(labels).Inc()
}

// Увеличение счетчика и установление
func (m *Metrics) IncConversionErrCount(labelOne, labelTwo, labelThree string) {
	labels := prom.Labels{"conv_type": labelOne, "entity": labelTwo, "result": labelThree}
	m.idConvErrCnt.MetricCollector.(*prom.CounterVec).With(labels).Inc()
}

func (m *Metrics) ObserveCustomDur(labelOne, labelTwo string, d time.Duration) {
	labels := prom.Labels{"label_one": labelOne, "label_two": labelTwo}
	m.idConvCnt.MetricCollector.(*prom.HistogramVec).With(labels).Observe(d.Seconds())
}

// извлекаем URL-параметр id и на основе результата увеличиваем значение счетчика метрики
func extractID(c echo.Context) (int, error) {
	metrics := c.Get(cKeyMetrics).(*Metrics) // получаем метрики из контекста по ключу
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		metrics.IncConversionErrCount("Atoi", strings.Split(c.Request().RequestURI, "/")[0], "err") // если ошибка, увеличиваем счетчик метрики, считающей ошибки при извлечении id
		return 0, err
	}

	metrics.IncConversionCount("Atoi", c.Request().RequestURI, c.Param("id")) // если нет ошибки, то увеличиваем счетчик метрики, считающей число корректных извлечений id
	return id, nil
}

func main() {
	r := echo.New()
	m := NewMetrics()                                                                                          // создаем и инициализируем объект, который хранит набор метрик
	p := prometheus.NewPrometheus("demo", nil, []*prometheus.Metric{m.idConvCnt, m.idConvErrCnt, m.customDur}) // создаем миддлварь prometheus для echo-роутера
	p.Use(r)                                                                                                   // применяем миддлварь для echo-роутера
	r.Use(m.AddCustomMetricsMiddleware)                                                                        // включаем миддлварь с кастомными метриками в цепь других миддлварей, которые стартуют после echo-роутера

	r.GET("/user/get/:id", func(c echo.Context) error {
		id, err := extractID(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, err.Error())
		}

		for _, u := range users {
			if u.ID == id {
				return c.JSON(http.StatusOK, u)
			}
		}

		return c.JSON(http.StatusNotFound, "user does not exist")
	})
	r.GET("/note/get/:id", func(c echo.Context) error {
		id, err := extractID(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, err.Error())
		}

		for _, n := range notes {
			if n.ID == id {
				return c.JSON(http.StatusOK, n)
			}
		}

		return c.JSON(http.StatusNotFound, "user does not exist")
	})
	r.GET("/user/get-list", func(c echo.Context) error {
		if users != nil {
			return c.JSON(http.StatusOK, users)
		}

		return c.JSON(http.StatusNotFound, "users does not exist")
	})
	r.GET("/note/get-list", func(c echo.Context) error {
		if users != nil {
			return c.JSON(http.StatusOK, notes)
		}

		return c.JSON(http.StatusNotFound, "notes does not exist")
	})

	// fmt.Println("Server is starting...")
	r.Logger.Fatal(r.Start(":8080"))
}
