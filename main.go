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
		getEntityReqDuration *prometheus.Metric
		// метрика - измеритель
		activeRequests     *prometheus.Metric
		activeUserRequests *prometheus.Metric
		activeNoteRequests *prometheus.Metric
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
		getEntityReqDuration: &prometheus.Metric{
			Name:        "get_entity_req_duration",
			Description: "Custom get entity request duration observations.",
			Type:        "histogram_vec",
			Args:        []string{"method", "entity"},
			Buckets:     prom.DefBuckets, // or your Buckets
		},
		activeRequests: &prometheus.Metric{
			Name:        "active_requests",
			Description: "Count of all active requests",
			Type:        "gauge_vec",
			Args:        []string{"entity", "id"},
		},
		activeUserRequests: &prometheus.Metric{
			Name:        "active_user_requests",
			Description: "Count of an active user requests",
			Type:        "gauge_vec",
			Args:        []string{"id"},
		},
		activeNoteRequests: &prometheus.Metric{
			Name:        "active_note_requests",
			Description: "Count of an active user requests",
			Type:        "gauge_vec",
			Args:        []string{"id"},
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

func (m *Metrics) ObserveGetEntityRequestDuration(methodLabel, entityLabel string, d time.Duration) {
	labels := prom.Labels{"method": methodLabel, "entity": entityLabel}
	m.getEntityReqDuration.MetricCollector.(*prom.HistogramVec).With(labels).Observe(float64(d.Milliseconds()))
}

func (m *Metrics) IncActiveRequestsGauge(metric *prometheus.Metric, entityLabel, idLabel string) {
	// Маппим значения лейблов из аргументов метода на соответствующие ключи
	labels := prom.Labels{"entity": entityLabel, "id": idLabel}
	// Добавляем мапу лейблов для метрики и увеличиваем значение счетчика метрики на 1
	metric.MetricCollector.(*prom.GaugeVec).With(labels).Inc()
}

func (m *Metrics) DecActiveRequestsGauge(metric *prometheus.Metric, entityLabel, idLabel string) {
	// Маппим значения лейблов из аргументов метода на соответствующие ключи
	labels := prom.Labels{"entity": entityLabel, "id": idLabel}
	// Добавляем мапу лейблов для метрики и увеличиваем значение счетчика метрики на 1
	metric.MetricCollector.(*prom.GaugeVec).With(labels).Dec()
}

func (m *Metrics) IncActiveUserRequestsGauge(metric *prometheus.Metric, idLabel string) {
	// Маппим значения лейблов из аргументов метода на соответствующие ключи
	labels := prom.Labels{"id": idLabel}
	// Добавляем мапу лейблов для метрики и увеличиваем значение счетчика метрики на 1
	metric.MetricCollector.(*prom.GaugeVec).With(labels).Inc()
}

func (m *Metrics) DecActiveUserRequestsGauge(metric *prometheus.Metric, idLabel string) {
	// Маппим значения лейблов из аргументов метода на соответствующие ключи
	labels := prom.Labels{"id": idLabel}
	// Добавляем мапу лейблов для метрики и увеличиваем значение счетчика метрики на 1
	metric.MetricCollector.(*prom.GaugeVec).With(labels).Dec()
}

func (m *Metrics) IncActiveNoteRequestsGauge(metric *prometheus.Metric, idLabel string) {
	// Маппим значения лейблов из аргументов метода на соответствующие ключи
	labels := prom.Labels{"id": idLabel}
	// Добавляем мапу лейблов для метрики и увеличиваем значение счетчика метрики на 1
	metric.MetricCollector.(*prom.GaugeVec).With(labels).Inc()
}

func (m *Metrics) DecActiveNoteRequestsGauge(metric *prometheus.Metric, idLabel string) {
	// Маппим значения лейблов из аргументов метода на соответствующие ключи
	labels := prom.Labels{"id": idLabel}
	// Добавляем мапу лейблов для метрики и увеличиваем значение счетчика метрики на 1
	metric.MetricCollector.(*prom.GaugeVec).With(labels).Dec()
}

// Извлекаем URL-параметр id и на основе результата увеличиваем значение счетчика метрики
func extractID(c echo.Context, metrics *Metrics) (int, error) {
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
	// metrics.ObserveCustomDur("l1", "l2", time.Second*10)
	return id, nil
}

func main() {
	// Создаем роутер
	r := echo.New()
	// Создаем и инициализируем объект, который хранит набор метрик
	m := NewMetrics()
	// Создаем миддлварь prometheus для echo-роутера
	p := prometheus.NewPrometheus("demo", nil, []*prometheus.Metric{
		m.idConvCnt,
		m.idConvErrCnt,
		m.getEntityReqDuration,
		m.activeRequests,
		m.activeUserRequests,
		m.activeNoteRequests,
	})
	// Добавляем миддлварь для echo-роутера
	p.Use(r)
	// Включаем миддлварь с кастомными метриками в цепь других миддлварей, которые стартуют после echo-роутера
	r.Use(m.AddCustomMetricsMiddleware)

	//ENDPOINTS

	//Get user by ID
	r.GET("/user/get/:id", func(c echo.Context) error {
		// Получаем метрики из контекста по ключу
		metrics := c.Get(cKeyMetrics).(*Metrics)
		metrics.IncActiveRequestsGauge(metrics.activeRequests, "user", c.Param("id"))
		metrics.IncActiveUserRequestsGauge(metrics.activeUserRequests, c.Param("id"))
		start := time.Now()

		id, err := extractID(c, metrics)
		if err != nil {
			metrics.ObserveGetEntityRequestDuration(http.MethodGet, "user", time.Since(start))
			metrics.DecActiveRequestsGauge(metrics.activeRequests, "user", c.Param("id"))
			metrics.DecActiveUserRequestsGauge(metrics.activeUserRequests, c.Param("id"))
			return c.JSON(http.StatusBadRequest, err.Error())
		}

		for _, u := range users {
			if u.ID == id {
				metrics.ObserveGetEntityRequestDuration(http.MethodGet, "user", time.Since(start))
				metrics.DecActiveRequestsGauge(metrics.activeRequests, "user", c.Param("id"))
				metrics.DecActiveUserRequestsGauge(metrics.activeUserRequests, c.Param("id"))
				return c.JSON(http.StatusOK, u)
			}
		}

		metrics.ObserveGetEntityRequestDuration(http.MethodGet, "user", time.Since(start))
		metrics.DecActiveRequestsGauge(metrics.activeRequests, "user", c.Param("id"))
		metrics.DecActiveUserRequestsGauge(metrics.activeUserRequests, c.Param("id"))
		return c.JSON(http.StatusNotFound, "user does not exist")
	})

	//Get note by ID
	r.GET("/note/get/:id", func(c echo.Context) error {
		// Получаем метрики из контекста по ключу
		metrics := c.Get(cKeyMetrics).(*Metrics)
		metrics.IncActiveRequestsGauge(metrics.activeRequests, "note", c.Param("id"))
		metrics.IncActiveUserRequestsGauge(metrics.activeUserRequests, c.Param("id"))
		start := time.Now()

		id, err := extractID(c, metrics)
		if err != nil {
			metrics.ObserveGetEntityRequestDuration(http.MethodGet, "note", time.Since(start))
			metrics.DecActiveRequestsGauge(metrics.activeRequests, "note", c.Param("id"))
			metrics.DecActiveUserRequestsGauge(metrics.activeUserRequests, c.Param("id"))
			return c.JSON(http.StatusBadRequest, err.Error())
		}

		for _, n := range notes {
			if n.ID == id {
				metrics.ObserveGetEntityRequestDuration(http.MethodGet, "note", time.Since(start))
				metrics.DecActiveRequestsGauge(metrics.activeRequests, "note", c.Param("id"))
				metrics.DecActiveUserRequestsGauge(metrics.activeUserRequests, c.Param("id"))
				return c.JSON(http.StatusOK, n)
			}
		}

		metrics.ObserveGetEntityRequestDuration(http.MethodGet, "note", time.Since(start))
		metrics.DecActiveRequestsGauge(metrics.activeRequests, "note", c.Param("id"))
		metrics.DecActiveUserRequestsGauge(metrics.activeUserRequests, c.Param("id"))
		return c.JSON(http.StatusNotFound, "note does not exist")
	})

	r.Logger.Fatal(r.Start(":8080"))
}
