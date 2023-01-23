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

	//Структура, хранящая набор метрик
	Metrics struct {
		// метрика - счетчик
		idConvCnt *prometheus.Metric
		// метрика - счетчик
		idConvErrCnt *prometheus.Metric
		// метрика - гистограмма
		customDur *prometheus.Metric
	}
)

var (
	// Список пользователей
	users = []*User{
		{ID: 1, Name: "Sam", Age: 15},
		{ID: 2, Name: "John", Age: 22, IsAdult: true},
		{ID: 3, Name: "Henrik", Age: 39, IsAdult: true},
	}
	// Список заметок
	notes = []*Note{
		{ID: 1, Title: "Homework", Text: "Math"},
		{ID: 2, Title: "Game info", Text: "Developer: Arkane Studios"},
		{ID: 3, Title: "Friend's email", Text: "friendsemail@example.com"},
	}
)

// Функция - конструктор для создания и инициализации структуры Metrics
func NewMetrics() *Metrics {
	return &Metrics{
		// Метрика, считающая кол - во удачных конвертаций URL - параметра id из строки в целое число
		idConvCnt: &prometheus.Metric{
			// Название метрики
			Name: "conversions_count",
			// Описание метрики
			Description: "id URL param conversions count",
			// Тип метрики (счетчик с аргументами (лейблами))
			Type: "counter_vec",
			// Аргументы (ключи) для лейблов, к которым будут маппиться значения, в зависимости от ситуации
			Args: []string{"conv_type", "entity", "result"},
		},
		// Метрика, считающая кол - во НЕудачных конвертаций URL - параметра id из строки в целое число
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
		// Добавляем указатель на структуру Metrics в контекст
		c.Set(cKeyMetrics, m)
		return next(c)
	}
}

// Маппинг лейблов и увеличение счетчика
func (m *Metrics) IncCounter(metric *prometheus.Metric, labelOne, labelTwo, labelThree string) {
	// Маппим значения лейблов из аргументов метода на соответствующие ключи
	labels := prom.Labels{"conv_type": labelOne, "entity": labelTwo, "result": labelThree}
	// Добавляем мапу лейблов для метрики и увеличиваем значение счетчика метрики на 1
	metric.MetricCollector.(*prom.CounterVec).With(labels).Inc()
}

func (m *Metrics) ObserveCustomDur(labelOne, labelTwo string, d time.Duration) {
	labels := prom.Labels{"label_one": labelOne, "label_two": labelTwo}
	m.customDur.MetricCollector.(*prom.HistogramVec).With(labels).Observe(d.Seconds())
}

// Извлекаем URL-параметр id и на основе результата увеличиваем значение счетчика метрики
func extractID(c echo.Context) (int, error) {
	// Получаем метрики из контекста по ключу
	metrics := c.Get(cKeyMetrics).(*Metrics)
	// Достаем id из URL и преобразовываем в число
	id, err := strconv.Atoi(c.Param("id"))
	// Получаем название сущности из URL
	entity := strings.Split(c.Request().RequestURI[1:], "/")[0]
	if err != nil {
		// Если ошибка, увеличиваем счетчик метрики, считающей ошибки
		metrics.IncCounter(metrics.idConvErrCnt, "Atoi", entity, "err")
		return 0, err
	}

	// Если нет ошибки, то увеличиваем счетчик метрики, считающей число корректных извлечений id
	metrics.IncCounter(metrics.idConvCnt, "Atoi", entity, c.Param("id"))
	metrics.ObserveCustomDur("l1", "l2", time.Second*10)
	return id, nil
}

func main() {
	// Создаем роутер
	r := echo.New()
	// Создаем и инициализируем объект, который хранит набор метрик
	m := NewMetrics()
	// Создаем миддлварь prometheus для echo-роутера
	p := prometheus.NewPrometheus("demo", nil, []*prometheus.Metric{m.idConvCnt, m.idConvErrCnt, m.customDur})
	// Добавляем миддлварь для echo-роутера
	p.Use(r)
	// Включаем миддлварь с кастомными метриками в цепь других миддлварей, которые стартуют после echo-роутера
	r.Use(m.AddCustomMetricsMiddleware)

	//Enpoints

	//Get user by ID
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

	//Get note by ID
	r.GET("/note/get/:id", func(c echo.Context) error {
		start := time.Now()
		id, err := extractID(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, err.Error())
		}

		for _, n := range notes {
			if n.ID == id {
				return c.JSON(http.StatusOK, n)
			}
		}

		return c.JSON(http.StatusNotFound, "note does not exist")
	})

	r.Logger.Fatal(r.Start(":8080"))
}
