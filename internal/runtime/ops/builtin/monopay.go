package builtin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/webitel/flow_manager/internal/runtime/ops"
)

const (
	monoCreateUrl                = "https://api.monobank.ua/api/merchant/invoice/create"
	monoCancelUrl                = "https://api.monobank.ua/api/merchant/invoice/cancel"
	monoInvalidateUrl            = "https://api.monobank.ua/api/merchant/invoice/remove"
	monoSuccessfulPaymentInfoUrl = "https://api.monobank.ua/api/merchant/invoice/payment-info"
	monoStatusUrl                = "https://api.monobank.ua/api/merchant/invoice/status"

	monoDefaultValidity = 3600
)

type monopayOp struct{}

// MonopayOp returns the native monoPay op (no external deps — pure HTTP).
func MonopayOp() ops.Op { return monopayOp{} }

func (monopayOp) Kind() ops.OpKind { return ops.OpKindSync }

func (o monopayOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	token, _ := in.Node.Args["token"].(string)
	if token == "" {
		return ops.OpOutput{}, fmt.Errorf("monoPay: token is required")
	}

	invoiceRaw, _ := in.Node.Args["invoice"].(map[string]any)
	action, _ := invoiceRaw["action"].(string)
	bodyRaw, _ := invoiceRaw["body"].(map[string]any)

	// expand all string values in body before decoding into typed structs
	expanded := expandMonoBody(bodyRaw, in.Variables, in.GlobalVar)

	switch action {
	case "create":
		var data monoCreateBody
		if err := jsonRoundtrip(expanded, &data); err != nil {
			return ops.OpOutput{}, fmt.Errorf("monoPay create: %w", err)
		}
		if err := data.normalize(); err != nil {
			return ops.OpOutput{}, err
		}
		res, err := monoCreatePayment(ctx, data, token)
		if err != nil {
			return ops.OpOutput{}, fmt.Errorf("monoPay create: %w", err)
		}
		return ops.OpOutput{SetVars: map[string]string{data.SetVar: string(res)}}, nil

	case "remove":
		var data monoInvalidateBody
		if err := jsonRoundtrip(expanded, &data); err != nil {
			return ops.OpOutput{}, fmt.Errorf("monoPay remove: %w", err)
		}
		if err := data.normalize(); err != nil {
			return ops.OpOutput{}, err
		}
		if err := monoInvalidatePayment(ctx, data, token); err != nil {
			return ops.OpOutput{}, fmt.Errorf("monoPay remove: %w", err)
		}
		return ops.OpOutput{}, nil

	case "cancel":
		var data monoCancelBody
		if err := jsonRoundtrip(expanded, &data); err != nil {
			return ops.OpOutput{}, fmt.Errorf("monoPay cancel: %w", err)
		}
		if err := data.normalize(); err != nil {
			return ops.OpOutput{}, err
		}
		res, err := monoCancelPayment(ctx, data, token)
		if err != nil {
			return ops.OpOutput{}, fmt.Errorf("monoPay cancel: %w", err)
		}
		return ops.OpOutput{SetVars: map[string]string{data.SetVar: string(res)}}, nil

	case "status":
		var data monoGetStatusBody
		if err := jsonRoundtrip(expanded, &data); err != nil {
			return ops.OpOutput{}, fmt.Errorf("monoPay status: %w", err)
		}
		if err := data.normalize(); err != nil {
			return ops.OpOutput{}, err
		}
		res, err := monoGetPaymentState(ctx, data, token)
		if err != nil {
			return ops.OpOutput{}, fmt.Errorf("monoPay status: %w", err)
		}
		return ops.OpOutput{SetVars: map[string]string{data.SetVar: string(res)}}, nil

	case "payment_info":
		var data monoGetSuccessfulPaymentInfoBody
		if err := jsonRoundtrip(expanded, &data); err != nil {
			return ops.OpOutput{}, fmt.Errorf("monoPay payment_info: %w", err)
		}
		if err := data.normalize(); err != nil {
			return ops.OpOutput{}, err
		}
		res, err := monoGetSuccessfulPaymentInfo(ctx, data, token)
		if err != nil {
			return ops.OpOutput{}, fmt.Errorf("monoPay payment_info: %w", err)
		}
		return ops.OpOutput{SetVars: map[string]string{data.SetVar: string(res)}}, nil

	default:
		return ops.OpOutput{}, fmt.Errorf("monoPay: unknown action %q", action)
	}
}

// expandMonoBody expands variable references in string values of the body map.
func expandMonoBody(body map[string]any, vars map[string]string, globalVar func(string) string) map[string]any {
	out := make(map[string]any, len(body))
	for k, v := range body {
		if s, ok := v.(string); ok {
			out[k] = ops.ExpandStr(s, vars, globalVar)
		} else {
			out[k] = v
		}
	}
	return out
}

// jsonRoundtrip marshals src to JSON then unmarshals into dst.
func jsonRoundtrip(src any, dst any) error {
	b, err := json.Marshal(src)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, dst)
}

// --- HTTP helpers ---

func monoCreatePayment(ctx context.Context, body monoCreateBody, token string) ([]byte, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, monoCreateUrl, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Token", token)
	req.Header.Set("X-Cms", "Webitel")
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return monoReadResponse(resp)
}

func monoCancelPayment(ctx context.Context, body monoCancelBody, token string) ([]byte, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, monoCancelUrl, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Token", token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return monoReadResponse(resp)
}

func monoInvalidatePayment(ctx context.Context, body monoInvalidateBody, token string) error {
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, monoInvalidateUrl, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("X-Token", token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var monoErr monoError
	if jsonErr := json.Unmarshal(raw, &monoErr); jsonErr != nil {
		return jsonErr
	}
	if monoErr.isValid() {
		return monoFormatError(monoErr)
	}
	return nil
}

func monoGetPaymentState(ctx context.Context, body monoGetStatusBody, token string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, monoStatusUrl, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Token", token)
	q := req.URL.Query()
	q.Add("invoiceId", body.InvoiceId)
	req.URL.RawQuery = q.Encode()
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return monoReadResponse(resp)
}

func monoGetSuccessfulPaymentInfo(ctx context.Context, body monoGetSuccessfulPaymentInfoBody, token string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, monoSuccessfulPaymentInfoUrl, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Token", token)
	q := req.URL.Query()
	q.Add("invoiceId", body.InvoiceId)
	req.URL.RawQuery = q.Encode()
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return monoReadResponse(resp)
}

func monoReadResponse(resp *http.Response) ([]byte, error) {
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var monoErr monoError
	if jsonErr := json.Unmarshal(raw, &monoErr); jsonErr != nil {
		return nil, jsonErr
	}
	if monoErr.ErrCode != "" {
		return nil, monoFormatError(monoErr)
	}
	return raw, nil
}

func monoFormatError(e monoError) error {
	return errors.New(fmt.Sprintf("mono: Code=%s, Error: %s", e.ErrCode, e.ErrText))
}

// --- Types ---

type monoCreateBody struct {
	Amount      int    `json:"amount,omitempty"`
	Ccy         int    `json:"ccy,omitempty"`
	PaymentType string `json:"paymentType,omitempty"`
	RedirectUrl string `json:"redirectUrl,omitempty"`
	ValidFor    int    `json:"validity,omitempty"`
	WebHookUrl  string `json:"webHookUrl,omitempty"`
	SetVar      string `json:"setVar"`
}

func (c *monoCreateBody) normalize() error {
	if c.ValidFor == 0 {
		c.ValidFor = monoDefaultValidity
	}
	if c.SetVar == "" {
		return errors.New("monoPay create: setVar is required")
	}
	switch c.PaymentType {
	case "debit", "hold":
	default:
		return fmt.Errorf("monoPay create: unknown paymentType %q", c.PaymentType)
	}
	return nil
}

type monoGetStatusBody struct {
	InvoiceId string `json:"invoiceId,omitempty"`
	SetVar    string `json:"setVar"`
}

func (c *monoGetStatusBody) normalize() error {
	if c.SetVar == "" {
		return errors.New("monoPay status: setVar is required")
	}
	if c.InvoiceId == "" {
		return errors.New("monoPay status: invoiceId is required")
	}
	return nil
}

type monoInvalidateBody struct {
	InvoiceId string `json:"invoiceId,omitempty"`
}

func (c *monoInvalidateBody) normalize() error {
	if c.InvoiceId == "" {
		return errors.New("monoPay remove: invoiceId is required")
	}
	return nil
}

type monoCancelBody struct {
	InvoiceId string `json:"invoiceId,omitempty"`
	ExtRef    string `json:"extRef,omitempty"`
	Amount    int    `json:"amount,omitempty"`
	SetVar    string `json:"setVar,omitempty"`
}

func (c *monoCancelBody) normalize() error {
	if c.InvoiceId == "" {
		return errors.New("monoPay cancel: invoiceId is required")
	}
	if c.SetVar == "" {
		return errors.New("monoPay cancel: setVar is required")
	}
	return nil
}

type monoGetSuccessfulPaymentInfoBody struct {
	InvoiceId string `json:"invoiceId,omitempty"`
	SetVar    string `json:"setVar,omitempty"`
}

func (c *monoGetSuccessfulPaymentInfoBody) normalize() error {
	if c.InvoiceId == "" {
		return errors.New("monoPay payment_info: invoiceId is required")
	}
	if c.SetVar == "" {
		return errors.New("monoPay payment_info: setVar is required")
	}
	return nil
}

type monoError struct {
	ErrCode string `json:"errCode,omitempty"`
	ErrText string `json:"errText,omitempty"`
}

func (e monoError) isValid() bool { return e.ErrCode != "" && e.ErrText != "" }
