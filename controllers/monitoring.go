package controllers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

type PromQLMonitorController struct {
	logger     *logrus.Logger
	promAPIURL string // URL Prometheus API, например "http://prometheus:9090/api/v1"
}

func NewPromQLMonitorController(logger *logrus.Logger, apiURL string) *PromQLMonitorController {
	return &PromQLMonitorController{
		logger:     logger,
		promAPIURL: apiURL,
	}
}

// ValidateCheck проверяет корректность параметров запроса PromQL
func (p *PromQLMonitorController) ValidateCheck(monitorMeta map[string]string) error {
	query := monitorMeta["query"]
	if query == "" {
		return fmt.Errorf("promql query is required")
	}
	return nil
}

// ValidateMonitoring — здесь можно проверить конфиг мониторинга, например, таймауты, повторные попытки и т.д.
// Для упрощения — просто возвращаем nil
func (p *PromQLMonitorController) ValidateMonitoring(config map[string]string) error {
	return nil
}

// RunCheck выполняет запрос к Prometheus API и анализирует результат
func (p *PromQLMonitorController) RunCheck(monitorMeta map[string]string) error {
	query := monitorMeta["query"]
	timeoutStr := monitorMeta["timeout"] // например, "5s"
	timeout := 10 * time.Second
	if timeoutStr != "" {
		if d, err := time.ParseDuration(timeoutStr); err == nil {
			timeout = d
		}
	}

	client := &http.Client{Timeout: timeout}

	url := fmt.Sprintf("%s/query?query=%s", p.promAPIURL, query)
	p.logger.Debugf("Running PromQL query: %s", url)

	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to query prometheus: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("prometheus API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result PrometheusQueryResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode prometheus response: %w", err)
	}

	if result.Status != "success" {
		return fmt.Errorf("prometheus query failed with status: %s", result.Status)
	}

	// Пример простой проверки: считаем, что успешный мониторинг — это когда query вернула не пустой набор данных
	if len(result.Data.Result) == 0 {
		return fmt.Errorf("promql query returned no data")
	}

	p.logger.Infof("PromQL monitoring check passed for query: %s", query)
	return nil
}

// CheckMonitoring — по сути alias для RunCheck, или можешь добавить дополнительную логику
func (p *PromQLMonitorController) CheckMonitoring(config map[string]string) error {
	return p.RunCheck(config)
}

//
// Вспомогательные типы для парсинга ответа Prometheus API
//

type PrometheusQueryResponse struct {
	Status string              `json:"status"`
	Data   PrometheusQueryData `json:"data"`
}

type PrometheusQueryData struct {
	ResultType string                  `json:"resultType"`
	Result     []PrometheusQueryResult `json:"result"`
}

type PrometheusQueryResult struct {
	Metric map[string]string `json:"metric"`
	Value  [2]interface{}    `json:"value"` // [ timestamp, value ]
}
