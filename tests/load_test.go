package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type loadTestConfig struct {
	BaseURL        string
	Requests       int
	Concurrency    int
	Timeout        time.Duration
	HighTopUp      int64
	LowTopUp       int64
	MixedHighRatio float64
}

type sendResponse struct {
	MessageID string `json:"messageId"`
	Status    string `json:"status"`
	Cost      int64  `json:"cost"`
}

func TestHighVolumeScenarios(t *testing.T) {
	if !loadTestEnabled() {
		t.Skip("set LOAD_TEST=1 to run load tests")
	}

	cfg := loadTestConfigFromEnv()
	client := &http.Client{Timeout: cfg.Timeout}

	suffix := strconv.FormatInt(time.Now().UnixNano(), 36)
	highCustomer := "load-high-" + suffix
	lowCustomer := "load-low-" + suffix

	createCustomer(t, client, cfg.BaseURL, highCustomer)
	createCustomer(t, client, cfg.BaseURL, lowCustomer)
	mustTopUp(t, client, cfg.BaseURL, highCustomer, cfg.HighTopUp)
	mustTopUp(t, client, cfg.BaseURL, lowCustomer, cfg.LowTopUp)

	standardCount := int(float64(cfg.Requests) * 0.7)
	if standardCount < 1 {
		standardCount = cfg.Requests
	}
	expressCount := cfg.Requests - standardCount
	if expressCount < 1 {
		expressCount = standardCount / 3
	}

	start := time.Now().UTC()

	t.Run("standard-bulk", func(t *testing.T) {
		result := bulkSend(t, client, cfg, highCustomer, "/api/sms", standardCount, false)
		if result.ok == 0 {
			t.Fatalf("expected accepted requests, got 0")
		}
	})

	t.Run("express-bulk", func(t *testing.T) {
		result := bulkSend(t, client, cfg, highCustomer, "/api/sms/express", expressCount, false)
		if result.ok == 0 {
			t.Fatalf("expected accepted requests, got 0")
		}
	})

	t.Run("insufficient-balance", func(t *testing.T) {
		smallCount := cfg.Requests / 5
		if smallCount < 20 {
			smallCount = 20
		}
		result := bulkSend(t, client, cfg, lowCustomer, "/api/sms", smallCount, true)
		if result.paymentRequired == 0 {
			t.Fatalf("expected payment required responses")
		}
	})

	t.Run("idempotency", func(t *testing.T) {
		clientRef := "ref-" + strconv.FormatInt(time.Now().UnixNano(), 36)
		toNumber := "+98912" + strconv.FormatInt(time.Now().UnixNano()%1000000, 10)
		msg1 := sendOnce(t, client, cfg.BaseURL, highCustomer, "/api/sms", toNumber, "idempotency", clientRef)
		msg2 := sendOnce(t, client, cfg.BaseURL, highCustomer, "/api/sms", toNumber, "idempotency", clientRef)
		if msg1.MessageID == "" || msg2.MessageID == "" {
			t.Fatalf("expected message ids for idempotency test")
		}
		if msg1.MessageID != msg2.MessageID {
			t.Fatalf("idempotency failed: %s != %s", msg1.MessageID, msg2.MessageID)
		}
	})

	t.Run("reporting", func(t *testing.T) {
		from := start.Add(-5 * time.Minute).Format(time.RFC3339)
		to := time.Now().UTC().Add(1 * time.Minute).Format(time.RFC3339)
		reportTo := fmt.Sprintf("+98914%06d", time.Now().UnixNano()%1000000)
		sendOnce(t, client, cfg.BaseURL, highCustomer, "/api/sms", reportTo, "reporting", "report-"+strconv.FormatInt(time.Now().UnixNano(), 36))
		query := fmt.Sprintf("/api/sms?from=%s&to=%s&toNumber=%s&limit=50", from, to, url.QueryEscape(reportTo))
		messages := listMessages(t, client, cfg.BaseURL, highCustomer, query)
		if len(messages) == 0 {
			t.Fatalf("expected non-empty report")
		}
	})

	t.Run("mixed-distribution", func(t *testing.T) {
		result := mixedSend(t, client, cfg, highCustomer, lowCustomer, "/api/sms", cfg.Requests/2)
		if result.ok == 0 {
			t.Fatalf("expected accepted requests, got 0")
		}
	})
}

type bulkResult struct {
	ok              int64
	paymentRequired int64
	other           int64
}

func bulkSend(t *testing.T, client *http.Client, cfg loadTestConfig, customerID string, path string, total int, allowPaymentRequired bool) bulkResult {
	t.Helper()

	var okCount int64
	var payCount int64
	var otherCount int64

	var wg sync.WaitGroup
	sem := make(chan struct{}, cfg.Concurrency)

	for i := 0; i < total; i++ {
		wg.Add(1)
		sem <- struct{}{}
		idx := i
		go func() {
			defer wg.Done()
			defer func() { <-sem }()

			toNumber := fmt.Sprintf("+98912%06d", idx%1000000)
			payload := map[string]string{
				"to":   toNumber,
				"text": "load-test-" + strconv.Itoa(idx),
			}

			status, _, err := postJSON(client, cfg.BaseURL+path, payload, map[string]string{
				"X-Customer-ID": customerID,
			})
			if err != nil {
				atomic.AddInt64(&otherCount, 1)
				return
			}

			switch status {
			case http.StatusAccepted:
				atomic.AddInt64(&okCount, 1)
			case http.StatusPaymentRequired:
				atomic.AddInt64(&payCount, 1)
			default:
				atomic.AddInt64(&otherCount, 1)
			}
		}()
	}

	wg.Wait()

	if otherCount > 0 {
		t.Fatalf("unexpected responses: %d", otherCount)
	}
	if !allowPaymentRequired && payCount > 0 {
		t.Fatalf("unexpected payment required responses: %d", payCount)
	}

	return bulkResult{ok: okCount, paymentRequired: payCount, other: otherCount}
}

func mixedSend(t *testing.T, client *http.Client, cfg loadTestConfig, highCustomer string, lowCustomer string, path string, total int) bulkResult {
	t.Helper()

	var okCount int64
	var payCount int64
	var otherCount int64

	var wg sync.WaitGroup
	sem := make(chan struct{}, cfg.Concurrency)
	for i := 0; i < total; i++ {
		wg.Add(1)
		sem <- struct{}{}
		idx := i
		go func() {
			defer wg.Done()
			defer func() { <-sem }()

			rng := rand.New(rand.NewSource(time.Now().UnixNano() + int64(idx)))
			customerID := lowCustomer
			if rng.Float64() < cfg.MixedHighRatio {
				customerID = highCustomer
			}

			endpoint := path
			if rng.Intn(2) == 0 {
				endpoint = "/api/sms/express"
			}

			toNumber := fmt.Sprintf("+98913%06d", idx%1000000)
			payload := map[string]string{
				"to":   toNumber,
				"text": "mixed-load-" + strconv.Itoa(idx),
			}

			status, _, err := postJSON(client, cfg.BaseURL+endpoint, payload, map[string]string{
				"X-Customer-ID": customerID,
			})
			if err != nil {
				atomic.AddInt64(&otherCount, 1)
				return
			}

			switch status {
			case http.StatusAccepted:
				atomic.AddInt64(&okCount, 1)
			case http.StatusPaymentRequired:
				atomic.AddInt64(&payCount, 1)
			default:
				atomic.AddInt64(&otherCount, 1)
			}
		}()
	}

	wg.Wait()

	if otherCount > 0 {
		t.Fatalf("unexpected responses: %d", otherCount)
	}

	return bulkResult{ok: okCount, paymentRequired: payCount, other: otherCount}
}

func sendOnce(t *testing.T, client *http.Client, baseURL string, customerID string, path string, toNumber string, text string, clientRef string) sendResponse {
	t.Helper()

	payload := map[string]string{
		"to":        toNumber,
		"text":      text,
		"clientRef": clientRef,
	}

	status, body, err := postJSON(client, baseURL+path, payload, map[string]string{
		"X-Customer-ID": customerID,
	})
	if err != nil {
		t.Fatalf("send request failed: %v", err)
	}
	if status != http.StatusAccepted {
		t.Fatalf("unexpected status: %d", status)
	}

	var resp sendResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("invalid response: %v", err)
	}

	return resp
}

func listMessages(t *testing.T, client *http.Client, baseURL string, customerID string, query string) []map[string]any {
	t.Helper()

	req, err := http.NewRequest(http.MethodGet, baseURL+query, nil)
	if err != nil {
		t.Fatalf("list request failed: %v", err)
	}
	req.Header.Set("X-Customer-ID", customerID)

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("list request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("unexpected list status: %d %s", resp.StatusCode, string(body))
	}

	var messages []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&messages); err != nil {
		t.Fatalf("invalid list response: %v", err)
	}

	return messages
}

func createCustomer(t *testing.T, client *http.Client, baseURL string, customerID string) {
	t.Helper()

	payload := map[string]string{
		"customerId": customerID,
		"name":       "Load Test " + customerID,
	}

	status, body, err := postJSON(client, baseURL+"/api/customers", payload, nil)
	if err != nil {
		t.Fatalf("create customer failed: %v", err)
	}
	if status != http.StatusCreated && status != http.StatusConflict {
		t.Fatalf("unexpected create status: %d %s", status, string(body))
	}
}

func mustTopUp(t *testing.T, client *http.Client, baseURL string, customerID string, amount int64) {
	t.Helper()

	payload := map[string]int64{"amount": amount}
	status, body, err := postJSON(client, baseURL+"/api/wallet/topup", payload, map[string]string{
		"X-Customer-ID": customerID,
	})
	if err != nil {
		t.Fatalf("topup failed: %v", err)
	}
	if status != http.StatusOK {
		t.Fatalf("unexpected topup status: %d %s", status, string(body))
	}
}

func postJSON(client *http.Client, url string, payload any, headers map[string]string) (int, []byte, error) {
	var body []byte
	if payload != nil {
		var err error
		body, err = json.Marshal(payload)
		if err != nil {
			return 0, nil, err
		}
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, err
	}

	return resp.StatusCode, respBody, nil
}

func loadTestConfigFromEnv() loadTestConfig {
	return loadTestConfig{
		BaseURL:        getEnv("LOAD_TEST_BASE_URL", "http://localhost:8080"),
		Requests:       getEnvInt("LOAD_TEST_REQUESTS", 500),
		Concurrency:    getEnvInt("LOAD_TEST_CONCURRENCY", 50),
		Timeout:        time.Duration(getEnvInt("LOAD_TEST_TIMEOUT_MS", 10000)) * time.Millisecond,
		HighTopUp:      getEnvInt64("LOAD_TEST_HIGH_TOPUP", 100000),
		LowTopUp:       getEnvInt64("LOAD_TEST_LOW_TOPUP", 5),
		MixedHighRatio: getEnvFloat("LOAD_TEST_MIXED_HIGH_RATIO", 0.8),
	}
}

func loadTestEnabled() bool {
	val := strings.ToLower(strings.TrimSpace(os.Getenv("LOAD_TEST")))
	return val == "1" || val == "true" || val == "yes"
}

func getEnv(key string, def string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return def
	}
	return value
}

func getEnvInt(key string, def int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return def
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return def
	}
	return parsed
}

func getEnvInt64(key string, def int64) int64 {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return def
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return def
	}
	return parsed
}

func getEnvFloat(key string, def float64) float64 {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return def
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return def
	}
	return parsed
}
