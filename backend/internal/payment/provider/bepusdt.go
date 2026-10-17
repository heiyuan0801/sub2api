package provider

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/Wei-Shaw/sub2api/internal/util/urlvalidator"
	"github.com/shopspring/decimal"
)

const (
	bepusdtMerchantID      = "1000"
	bepusdtHTTPTimeout     = 15 * time.Second
	bepusdtMaxPayloadSize  = 1 << 20
	bepusdtPaymentWaiting  = 1
	bepusdtPaymentPaid     = 2
	bepusdtPaymentExpired  = 3
	bepusdtPaymentCanceled = 4
	bepusdtPaymentPending  = 5
	bepusdtPaymentFailed   = 6
)

// Provider instances are recreated for each operation. Share the transport so
// their HTTPS connections are reused rather than leaving a new idle pool each time.
var bepusdtClient = func() *http.Client {
	transport := &http.Transport{}
	if defaultTransport, ok := http.DefaultTransport.(*http.Transport); ok {
		transport = defaultTransport.Clone()
	}
	// Validate and pin public addresses at dial time, without sending signed
	// merchant requests through environment-controlled proxies.
	transport.Proxy = nil
	transport.DialContext = bepusdtDialContext
	return &http.Client{
		Transport: transport,
		Timeout:   bepusdtHTTPTimeout,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}()

// BEpusdt uses the native signed JSON API, rather than the partial EasyPay
// compatibility interface. The native API identifies the merchant by apiToken;
// the fixed EasyPay PID is retained only as non-sensitive instance metadata.
type BEpusdt struct {
	instanceID string
	config     map[string]string
	httpClient *http.Client
}

func NewBEpusdt(instanceID string, config map[string]string) (*BEpusdt, error) {
	cfg := cloneStringMap(config)
	for _, key := range []string{"apiBase", "apiToken"} {
		cfg[key] = strings.TrimSpace(cfg[key])
		if cfg[key] == "" {
			return nil, fmt.Errorf("bepusdt config missing required key: %s", key)
		}
	}
	if pid := strings.TrimSpace(cfg["pid"]); pid != "" && pid != bepusdtMerchantID {
		return nil, errors.New("bepusdt pid must be 1000")
	}
	cfg["pid"] = bepusdtMerchantID
	base, err := url.Parse(cfg["apiBase"])
	if err != nil || base.Scheme != "https" || base.Host == "" || base.User != nil || base.RawQuery != "" || base.ForceQuery || base.Fragment != "" {
		return nil, errors.New("bepusdt apiBase must be an HTTPS base URL without credentials, query or fragment")
	}
	if _, err := urlvalidator.ValidateHTTPSURL(cfg["apiBase"], urlvalidator.ValidationOptions{}); err != nil {
		return nil, errors.New("bepusdt apiBase must use a public HTTPS host")
	}
	base.Path = strings.TrimRight(base.Path, "/")
	base.RawPath = ""
	cfg["apiBase"] = base.String()
	currency, err := bepusdtCurrency(cfg["currency"])
	if err != nil {
		return nil, err
	}
	cfg["currency"] = currency
	cfg["tradeType"] = strings.TrimSpace(cfg["tradeType"])
	cfg["currencies"] = strings.TrimSpace(cfg["currencies"])
	if cfg["tradeType"] != "" && !bepusdtIdentifier(cfg["tradeType"]) {
		return nil, errors.New("bepusdt tradeType contains invalid characters")
	}
	return &BEpusdt{
		instanceID: instanceID,
		config:     cfg,
		httpClient: bepusdtClient,
	}, nil
}

func (b *BEpusdt) Name() string        { return "BEpusdt" }
func (b *BEpusdt) ProviderKey() string { return payment.TypeBEpusdt }
func (b *BEpusdt) SupportedTypes() []payment.PaymentType {
	return []payment.PaymentType{payment.TypeBEpusdt}
}

func (b *BEpusdt) MerchantIdentityMetadata() map[string]string {
	return map[string]string{"pid": bepusdtMerchantID, "currency": b.config["currency"]}
}

func (b *BEpusdt) CreatePayment(ctx context.Context, req payment.CreatePaymentRequest) (*payment.CreatePaymentResponse, error) {
	if req.PaymentType != payment.TypeBEpusdt {
		return nil, errors.New("bepusdt create payment: unsupported payment method")
	}
	if !bepusdtIdentifier(req.OrderID) {
		return nil, errors.New("bepusdt create payment: invalid order id")
	}
	amount, err := bepusdtAmount(json.RawMessage(strconv.Quote(strings.TrimSpace(req.Amount))))
	if err != nil {
		return nil, errors.New("bepusdt create payment: invalid positive fixed amount")
	}
	if parsed := decimal.NewFromFloat(amount); !parsed.Equal(parsed.Round(int32(payment.CurrencyMaxFractionDigits(b.config["currency"])))) {
		return nil, errors.New("bepusdt create payment: amount precision exceeds the fiat currency limit")
	}
	notifyURL, returnURL := strings.TrimSpace(req.NotifyURL), strings.TrimSpace(req.ReturnURL)
	if notifyURL == "" {
		notifyURL = strings.TrimSpace(b.config["notifyUrl"])
	}
	if returnURL == "" {
		returnURL = strings.TrimSpace(b.config["returnUrl"])
	}
	if !bepusdtCallbackURL(notifyURL) || !bepusdtCallbackURL(returnURL) {
		return nil, errors.New("bepusdt create payment: valid notify and return URLs are required")
	}
	payload := map[string]any{
		"order_id": req.OrderID, "amount": amount, "fiat": b.config["currency"],
		"notify_url": notifyURL, "redirect_url": returnURL,
	}
	if req.TimeoutSeconds != 0 {
		// Native code silently falls back to its own default outside this range.
		// Reject it here so the cashier does not outlive the local order policy.
		if req.TimeoutSeconds < 180 || req.TimeoutSeconds > 3600 {
			return nil, errors.New("bepusdt order timeout must be between 180 and 3600 seconds")
		}
		payload["timeout"] = float64(req.TimeoutSeconds)
	}
	if req.Subject != "" {
		payload["name"] = req.Subject
	}
	endpoint := "/api/v1/order/create-order"
	if tradeType := b.config["tradeType"]; tradeType != "" {
		endpoint = "/api/v1/order/create-transaction"
		payload["trade_type"] = tradeType
	} else if currencies := b.config["currencies"]; currencies != "" {
		payload["currencies"] = currencies
	}
	payload["signature"] = bepusdtSign(payload, b.config["apiToken"])
	var data bepusdtOrderData
	if err := b.post(ctx, endpoint, payload, &data); err != nil {
		return nil, fmt.Errorf("bepusdt create payment: %w", err)
	}
	if !bepusdtIdentifier(data.TradeID) || data.OrderID != req.OrderID {
		return nil, errors.New("bepusdt create payment: upstream order identifiers do not match")
	}
	returnedAmount, err := bepusdtAmount(data.Amount)
	if err != nil || !bepusdtAmountsEqual(returnedAmount, amount) {
		return nil, errors.New("bepusdt create payment: upstream amount does not match")
	}
	if strings.ToUpper(strings.TrimSpace(data.Fiat)) != b.config["currency"] {
		return nil, errors.New("bepusdt create payment: upstream fiat currency does not match")
	}
	payURL, err := url.Parse(strings.TrimSpace(data.PaymentURL))
	if err != nil || payURL.Host == "" || payURL.User != nil {
		return nil, errors.New("bepusdt create payment: invalid upstream checkout URL")
	}
	// CreateOrder in upstream BEpusdt checks only TLS on the origin request.
	// Behind a TLS reverse proxy it can return HTTP for its own HTTPS host.
	baseURL, _ := url.Parse(b.config["apiBase"])
	if payURL.Scheme == "http" && strings.EqualFold(payURL.Host, baseURL.Host) {
		payURL.Scheme = "https"
	}
	if payURL.Scheme != "https" {
		return nil, errors.New("bepusdt create payment: checkout URL must use HTTPS")
	}
	if _, err := urlvalidator.ValidateHTTPSURL(payURL.String(), urlvalidator.ValidationOptions{}); err != nil {
		return nil, errors.New("bepusdt create payment: invalid upstream checkout host")
	}
	return &payment.CreatePaymentResponse{
		TradeNo: data.TradeID, PayURL: payURL.String(), Currency: b.config["currency"],
	}, nil
}

type bepusdtOrderData struct {
	TradeID    string          `json:"trade_id"`
	OrderID    string          `json:"order_id"`
	Fiat       string          `json:"fiat"`
	Amount     json.RawMessage `json:"amount"`
	Money      json.RawMessage `json:"money"`
	Status     json.RawMessage `json:"status"`
	PaymentURL string          `json:"payment_url"`
}

// QueryOrder uses the cashier info endpoint. BEpusdt may bind a cashier order
// to the payer's browser fingerprint, in which case this server-side request
// fails closed. Signed webhooks remain the authoritative completion path;
// implementing NotificationConfirmer here would block those valid webhooks.
func (b *BEpusdt) QueryOrder(ctx context.Context, tradeNo string) (*payment.QueryOrderResponse, error) {
	if !bepusdtIdentifier(tradeNo) {
		return nil, errors.New("bepusdt query order: invalid trade id")
	}
	var data bepusdtOrderData
	if err := b.post(ctx, "/api/v1/pay/info", map[string]any{"trade_id": tradeNo}, &data); err != nil {
		return nil, fmt.Errorf("bepusdt query order: %w", err)
	}
	if data.TradeID != tradeNo || !bepusdtIdentifier(data.OrderID) {
		return nil, errors.New("bepusdt query order: upstream order identifiers do not match")
	}
	amount, err := bepusdtAmount(data.Money)
	if err != nil {
		return nil, errors.New("bepusdt query order: invalid fiat amount")
	}
	currency := strings.ToUpper(strings.TrimSpace(data.Fiat))
	if currency != b.config["currency"] {
		return nil, errors.New("bepusdt query order: upstream fiat currency does not match")
	}
	status, err := bepusdtInteger(data.Status)
	if err != nil {
		return nil, errors.New("bepusdt query order: invalid status")
	}
	providerStatus := payment.ProviderStatusPending
	switch status {
	case bepusdtPaymentPaid:
		providerStatus = payment.ProviderStatusPaid
	case bepusdtPaymentExpired, bepusdtPaymentCanceled, bepusdtPaymentFailed:
		providerStatus = payment.ProviderStatusFailed
	case bepusdtPaymentWaiting, bepusdtPaymentPending:
	default:
		return nil, errors.New("bepusdt query order: unknown status")
	}
	return &payment.QueryOrderResponse{
		TradeNo: data.TradeID, Status: providerStatus, Amount: amount,
		Metadata: bepusdtMetadata(data.OrderID, data.TradeID, currency, status),
	}, nil
}

func (b *BEpusdt) VerifyNotification(_ context.Context, rawBody string, _ map[string]string) (*payment.PaymentNotification, error) {
	fields, err := bepusdtNotificationFields(rawBody)
	if err != nil {
		return nil, err
	}
	signature, ok := fields["signature"].(string)
	if !ok || len(signature) != md5.Size*2 {
		return nil, errors.New("bepusdt notification: missing or invalid signature")
	}
	if _, err := hex.DecodeString(signature); err != nil || !hmac.Equal([]byte(signature), []byte(bepusdtSign(fields, b.config["apiToken"]))) {
		return nil, errors.New("bepusdt notification: signature verification failed")
	}
	statusJSON, _ := json.Marshal(fields["status"])
	status, err := bepusdtInteger(statusJSON)
	if err != nil {
		return nil, errors.New("bepusdt notification: invalid status")
	}
	switch status {
	case bepusdtPaymentWaiting, bepusdtPaymentExpired, bepusdtPaymentCanceled, bepusdtPaymentPending, bepusdtPaymentFailed:
		return nil, nil
	case bepusdtPaymentPaid:
	default:
		return nil, errors.New("bepusdt notification: unknown status")
	}
	orderID, orderOK := fields["order_id"].(string)
	tradeID, tradeOK := fields["trade_id"].(string)
	if !orderOK || !tradeOK || !bepusdtIdentifier(orderID) || !bepusdtIdentifier(tradeID) {
		return nil, errors.New("bepusdt notification: missing or invalid order identifiers")
	}
	amountJSON, _ := json.Marshal(fields["amount"])
	amount, err := bepusdtAmount(amountJSON)
	if err != nil {
		return nil, errors.New("bepusdt notification: invalid positive fiat amount")
	}
	blockHash, _ := fields["block_transaction_id"].(string)
	// Upstream manual reconciliation permits an empty transaction hash. The
	// signed order/transaction IDs and fixed fiat amount still bind the payment.
	if len(blockHash) > 512 || strings.ContainsAny(blockHash, "&\r\n\x00") {
		return nil, errors.New("bepusdt notification: invalid transaction hash")
	}
	currency := b.config["currency"]
	// Current native callbacks omit fiat. The fixed fiat amount is matched
	// against the stored order by the service; honor signed fiat if supplied.
	if fiat, exists := fields["fiat"]; exists {
		value, ok := fiat.(string)
		if !ok || strings.ToUpper(strings.TrimSpace(value)) != currency {
			return nil, errors.New("bepusdt notification: fiat currency does not match")
		}
	}
	metadata := bepusdtMetadata(orderID, tradeID, currency, status)
	metadata["block_transaction_id"] = blockHash
	return &payment.PaymentNotification{
		TradeNo: tradeID, OrderID: orderID, Amount: amount,
		Status: payment.NotificationStatusSuccess, RawData: rawBody, Metadata: metadata,
	}, nil
}

func (b *BEpusdt) Refund(context.Context, payment.RefundRequest) (*payment.RefundResponse, error) {
	return nil, errors.New("bepusdt does not support automatic refunds; refund the on-chain payment manually")
}

func (b *BEpusdt) CancelPayment(ctx context.Context, tradeNo string) error {
	if !bepusdtIdentifier(tradeNo) {
		return errors.New("bepusdt cancel payment: invalid trade id")
	}
	payload := map[string]any{"trade_id": tradeNo}
	payload["signature"] = bepusdtSign(payload, b.config["apiToken"])
	var data struct {
		TradeID string `json:"trade_id"`
	}
	if err := b.post(ctx, "/api/v1/order/cancel-transaction", payload, &data); err != nil {
		return fmt.Errorf("bepusdt cancel payment: %w", err)
	}
	if data.TradeID != tradeNo {
		return errors.New("bepusdt cancel payment: upstream trade id does not match")
	}
	return nil
}

func (b *BEpusdt) post(ctx context.Context, endpoint string, payload map[string]any, result any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return errors.New("cannot encode upstream request")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, b.config["apiBase"]+endpoint, bytes.NewReader(body))
	if err != nil {
		return errors.New("cannot create upstream request")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := b.httpClient.Do(req)
	if err != nil {
		// Do not include gateway response bodies, signed requests or transport
		// errors containing URLs in logs or API errors.
		return errors.New("upstream request failed")
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("upstream returned HTTP %d", resp.StatusCode)
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, bepusdtMaxPayloadSize+1))
	if err != nil || len(raw) > bepusdtMaxPayloadSize {
		return errors.New("upstream response unreadable or too large")
	}
	var envelope struct {
		StatusCode json.RawMessage `json:"status_code"`
		Data       json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return errors.New("invalid upstream JSON response")
	}
	statusCode, err := bepusdtInteger(envelope.StatusCode)
	if err != nil || statusCode != http.StatusOK {
		return errors.New("upstream rejected the request (check configuration, payment methods and order state)")
	}
	if len(envelope.Data) == 0 || string(envelope.Data) == "null" || json.Unmarshal(envelope.Data, result) != nil {
		return errors.New("invalid upstream order data")
	}
	return nil
}

func bepusdtSign(fields map[string]any, token string) string {
	keys := make([]string, 0, len(fields))
	for key, value := range fields {
		if key == "signature" || value == nil {
			continue
		}
		if text, ok := value.(string); ok && text == "" {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		// BEpusdt unmarshals JSON into map[string]any before signing, so JSON
		// numbers use Go's float64 formatting, not their original JSON spelling.
		parts = append(parts, key+"="+fmt.Sprint(fields[key]))
	}
	sum := md5.Sum([]byte(strings.Join(parts, "&") + token))
	return hex.EncodeToString(sum[:])
}

// Decode once, rejecting duplicate keys and composite values before verifying
// the signature. Standard json.Unmarshal silently accepts duplicate fields.
func bepusdtNotificationFields(raw string) (map[string]any, error) {
	if len(raw) == 0 || len(raw) > bepusdtMaxPayloadSize {
		return nil, errors.New("bepusdt notification: invalid body size")
	}
	decoder := json.NewDecoder(strings.NewReader(raw))
	first, err := decoder.Token()
	if err != nil || first != json.Delim('{') {
		return nil, errors.New("bepusdt notification: expected JSON object")
	}
	fields := make(map[string]any)
	for decoder.More() {
		keyToken, err := decoder.Token()
		key, ok := keyToken.(string)
		if err != nil || !ok || !bepusdtIdentifier(key) {
			return nil, errors.New("bepusdt notification: invalid field name")
		}
		if _, exists := fields[key]; exists {
			return nil, errors.New("bepusdt notification: duplicate field")
		}
		var rawValue json.RawMessage
		if err := decoder.Decode(&rawValue); err != nil {
			return nil, errors.New("bepusdt notification: invalid JSON field")
		}
		var value any
		if err := json.Unmarshal(rawValue, &value); err != nil {
			return nil, errors.New("bepusdt notification: invalid JSON value")
		}
		switch typed := value.(type) {
		case nil, bool:
		case string:
			if strings.ContainsAny(typed, "&\x00") {
				return nil, errors.New("bepusdt notification: ambiguous field value")
			}
		case float64:
			original, err := decimal.NewFromString(string(rawValue))
			if err != nil || original.Exponent() < -308 || original.Exponent() > 308 || math.IsNaN(typed) || math.IsInf(typed, 0) || !original.Equal(decimal.NewFromFloat(typed)) {
				return nil, errors.New("bepusdt notification: imprecise numeric value")
			}
		default:
			return nil, errors.New("bepusdt notification: composite values are not supported")
		}
		fields[key] = value
	}
	if closing, err := decoder.Token(); err != nil || closing != json.Delim('}') {
		return nil, errors.New("bepusdt notification: malformed JSON object")
	}
	if _, err := decoder.Token(); err != io.EOF {
		return nil, errors.New("bepusdt notification: trailing JSON data")
	}
	return fields, nil
}

func bepusdtAmount(raw json.RawMessage) (float64, error) {
	value := strings.TrimSpace(string(raw))
	if strings.HasPrefix(value, "\"") {
		if err := json.Unmarshal(raw, &value); err != nil {
			return 0, errors.New("invalid amount")
		}
	}
	if len(value) == 0 || len(value) > 64 {
		return 0, errors.New("invalid amount")
	}
	floatAmount, err := strconv.ParseFloat(value, 64)
	if err != nil || floatAmount <= 0 || math.IsNaN(floatAmount) || math.IsInf(floatAmount, 0) {
		return 0, errors.New("invalid amount")
	}
	amount, err := decimal.NewFromString(value)
	if err != nil || !amount.IsPositive() || amount.Exponent() < -308 || amount.Exponent() > 308 {
		return 0, errors.New("invalid amount")
	}
	if !amount.Equal(decimal.NewFromFloat(floatAmount)) {
		return 0, errors.New("amount cannot be represented safely")
	}
	return floatAmount, nil
}

func bepusdtInteger(raw json.RawMessage) (int, error) {
	value := strings.TrimSpace(string(raw))
	if strings.HasPrefix(value, "\"") {
		if err := json.Unmarshal(raw, &value); err != nil {
			return 0, errors.New("invalid integer")
		}
	}
	return strconv.Atoi(value)
}

func bepusdtAmountsEqual(left, right float64) bool {
	return decimal.NewFromFloat(left).Equal(decimal.NewFromFloat(right))
}

func bepusdtMetadata(orderID, tradeID, currency string, status int) map[string]string {
	return map[string]string{
		"pid": bepusdtMerchantID, "currency": currency, "order_id": orderID,
		"out_trade_no": orderID, "trade_id": tradeID, "status": strconv.Itoa(status),
	}
}

func bepusdtIdentifier(value string) bool {
	if len(value) == 0 || len(value) > 256 {
		return false
	}
	for _, ch := range value {
		if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_' || ch == '-' || ch == '.') {
			return false
		}
	}
	return true
}

func bepusdtCurrency(raw string) (string, error) {
	currency := strings.ToUpper(strings.TrimSpace(raw))
	if currency == "" {
		currency = "CNY"
	}
	switch currency {
	case "CNY", "USD", "EUR", "GBP", "JPY":
		return currency, nil
	default:
		return "", errors.New("bepusdt currency must be CNY, USD, EUR, GBP or JPY")
	}
}

func bepusdtCallbackURL(raw string) bool {
	parsed, err := url.Parse(raw)
	return err == nil && (parsed.Scheme == "https" || parsed.Scheme == "http") && parsed.Host != "" && parsed.User == nil && parsed.Fragment == ""
}

func bepusdtDialContext(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, errors.New("invalid payment gateway address")
	}
	addresses, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil || len(addresses) == 0 {
		return nil, errors.New("payment gateway DNS resolution failed")
	}
	for _, addr := range addresses {
		if !addr.IP.IsGlobalUnicast() || urlvalidator.IsBlockedHost(addr.IP.String()) || bepusdtCarrierNAT(addr.IP) {
			return nil, errors.New("payment gateway resolved to a blocked address")
		}
	}
	dialer := &net.Dialer{Timeout: bepusdtHTTPTimeout, KeepAlive: 30 * time.Second}
	for _, addr := range addresses {
		conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(addr.IP.String(), port))
		if err == nil {
			return conn, nil
		}
	}
	return nil, errors.New("payment gateway connection failed")
}

func bepusdtCarrierNAT(ip net.IP) bool {
	ipv4 := ip.To4()
	return ipv4 != nil && ipv4[0] == 100 && ipv4[1] >= 64 && ipv4[1] <= 127
}
