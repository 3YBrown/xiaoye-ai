package payment

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"google-ai-proxy/internal/config"
	"google-ai-proxy/internal/db"
)

var httpClient = &http.Client{Timeout: 10 * time.Second}

const (
	linuxDoCreditAPIBase = "https://credit.linux.do/epay"
	linuxDoCreditSubmit  = linuxDoCreditAPIBase + "/pay/submit.php"
	linuxDoCreditQuery   = linuxDoCreditAPIBase + "/api.php"
)

// LinuxDoCreditProvider implements PaymentProvider for credit.linux.do EPay.
type LinuxDoCreditProvider struct{}

func NewLinuxDoCreditProvider() *LinuxDoCreditProvider {
	return &LinuxDoCreditProvider{}
}

func (p *LinuxDoCreditProvider) Name() string {
	return "linuxdo_credit"
}

// sign builds the MD5 sign string per EPay spec:
// 1. Filter empty values, exclude sign and sign_type
// 2. Sort keys by ASCII ascending
// 3. Concatenate k1=v1&k2=v2
// 4. Append secret key
// 5. MD5 lowercase hex
func sign(params map[string]string, secret string) string {
	keys := make([]string, 0, len(params))
	for k, v := range params {
		if k == "sign" || k == "sign_type" || v == "" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+"="+params[k])
	}
	raw := strings.Join(parts, "&") + secret
	return fmt.Sprintf("%x", md5.Sum([]byte(raw)))
}

func (p *LinuxDoCreditProvider) CreatePayment(order *db.PaymentOrder) (*PaymentResult, error) {
	pid := config.GetLinuxDoCreditPID()
	key := config.GetLinuxDoCreditKey()
	notifyURL := config.GetLinuxDoCreditNotifyURL()
	returnURL := config.GetLinuxDoCreditReturnURL()

	params := map[string]string{
		"pid":          pid,
		"type":         "epay",
		"out_trade_no": order.OrderNo,
		"notify_url":   notifyURL,
		"return_url":   returnURL,
		"name":         fmt.Sprintf("小野AI %s - %d diamonds", order.PlanName, order.Diamonds),
		"money":        order.Amount,
	}
	params["sign"] = sign(params, key)
	params["sign_type"] = "MD5"

	// Build submit URL
	u, _ := url.Parse(linuxDoCreditSubmit)
	q := u.Query()
	for k, v := range params {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()

	return &PaymentResult{PaymentURL: u.String()}, nil
}

func (p *LinuxDoCreditProvider) VerifyNotify(params map[string]string) (bool, error) {
	key := config.GetLinuxDoCreditKey()
	receivedSign := params["sign"]
	if receivedSign == "" {
		return false, fmt.Errorf("missing sign")
	}
	expected := sign(params, key)
	return receivedSign == expected, nil
}

func (p *LinuxDoCreditProvider) QueryOrder(order *db.PaymentOrder) (*QueryResult, error) {
	pid := config.GetLinuxDoCreditPID()
	key := config.GetLinuxDoCreditKey()

	// Query API uses pid + key directly, not signed
	u, _ := url.Parse(linuxDoCreditQuery)
	q := u.Query()
	q.Set("act", "order")
	q.Set("pid", pid)
	q.Set("key", key)
	q.Set("out_trade_no", order.OrderNo)
	u.RawQuery = q.Encode()

	resp, err := httpClient.Get(u.String())
	if err != nil {
		return nil, fmt.Errorf("query order failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read query response failed: %w", err)
	}

	var result struct {
		Code        int    `json:"code"`
		TradeNo     string `json:"trade_no"`
		TradeStatus string `json:"status"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse query response failed: %w", err)
	}

	if result.Code != 1 {
		return nil, fmt.Errorf("query returned code %d", result.Code)
	}

	return &QueryResult{
		TradeNo:     result.TradeNo,
		TradeStatus: result.TradeStatus,
	}, nil
}
