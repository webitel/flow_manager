package flow

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/webitel/flow_manager/model"
	"io"
	"net/http"
)

const (
	createUrl                = "https://api.monobank.ua/api/merchant/invoice/create"       // POST
	cancelUrl                = "https://api.monobank.ua/api/merchant/invoice/cancel"       // POST
	invalidateUrl            = "https://api.monobank.ua/api/merchant/invoice/remove"       // POST
	successfulPaymentInfoUrl = "https://api.monobank.ua/api/merchant/invoice/payment-info" // GET
	statusUrl                = "https://api.monobank.ua/api/merchant/invoice/status"       // GET

	defaultValidity = 3600
)

func (r *router) monopayHandler(ctx context.Context, scope *Flow, conn model.Connection, args interface{}) (model.Response, *model.AppError) {
	var (
		argv MonopayArgs
	)
	if err := scope.Decode(args, &argv); err != nil {
		return model.CallResponseError, err
	}
	if argv.Token == "" {
		return model.CallResponseError, model.NewAppError("", "flow.monopay.monopay_handler.check_args.error", nil, "mono: token is empty", http.StatusBadRequest)
	}
	switch argv.Invoice.Action {
	case "create":
		var data CreateBody
		appErr := scope.Decode(argv.Invoice.Body, &data)
		if appErr != nil {
			return model.CallResponseError, appErr
		}
		appErr = data.Normalize()
		if appErr != nil {
			return model.CallResponseError, appErr
		}
		res, err := CreatePayment(ctx, data, argv.Token)
		if err != nil {
			return model.CallResponseError, model.NewAppError("", "flow.monopay.monopay_handler.create_payment.error", nil, err.Error(), http.StatusInternalServerError)
		}

		return conn.Set(ctx, model.Variables{
			data.SetVar: string(res),
		})
		//		{
		//			"amount": 4200,
		//			"ccy": 980,
		//			"merchantPaymInfo": {
		//			"reference": "84d0070ee4e44667b31371d8f8813947",
		//				"destination": "Покупка щастя",
		//				"comment": "Покупка щастя",
		//				"customerEmails": [],
		//	"basketOrder": [
		//		{
		//		"name": "Табуретка",
		//		"qty": 2,
		//		"sum": 2100,
		//		"icon": "string",
		//		"unit": "шт.",
		//		"code": "d21da1c47f3c45fca10a10c32518bdeb",
		//		"barcode": "string",
		//		"header": "string",
		//		"footer": "string",
		//		"tax": [],
		//		"uktzed": "string",
		//		"discounts": [
		//		{
		//		"type": "DISCOUNT",
		//		"mode": "PERCENT",
		//		"value": "PERCENT"
		//		}
		//		]
		//		}
		//	]
		//	},
		//	"redirectUrl": "https://example.com/your/website/result/page",
		//		"webHookUrl": "https://example.com/mono/acquiring/webhook/maybesomegibberishuniquestringbutnotnecessarily",
		//		"validity": 3600,
		//		"paymentType": "debit",
		//		"qrId": "XJ_DiM4rTd5V",
		//		"code": "0a8637b3bccb42aa93fdeb791b8b58e9",
		//		"saveCardData": {
		//		"saveCard": true,
		//			"walletId": "69f780d841a0434aa535b08821f4822c"
		//	}
		//}
	case "remove":
		var data InvalidateBody
		appErr := scope.Decode(argv.Invoice.Body, &data)
		if appErr != nil {
			return model.CallResponseError, appErr
		}
		appErr = data.Normalize()
		if appErr != nil {
			return model.CallResponseError, appErr
		}
		err := InvalidatePayment(ctx, data, argv.Token)
		if err != nil {
			return model.CallResponseError, model.NewAppError("", "flow.monopay.monopay_handler.invalidate_payment.error", nil, err.Error(), http.StatusInternalServerError)
		}
		return model.CallResponseOK, nil
	case "cancel":
		var data CancelBody
		appErr := scope.Decode(argv.Invoice.Body, &data)
		if appErr != nil {
			return model.CallResponseError, appErr
		}
		appErr = data.Normalize()
		if appErr != nil {
			return model.CallResponseError, appErr
		}
		res, err := CancelPayment(ctx, data, argv.Token)
		if err != nil {
			return model.CallResponseError, model.NewAppError("", "flow.monopay.monopay_handler.cancel_payment.error", nil, err.Error(), http.StatusInternalServerError)
		}
		return conn.Set(ctx, model.Variables{
			data.SetVar: string(res),
		})
		//	{
		//		"invoiceId": "p2_9ZgpZVsl3",
		//		"extRef": "635ace02599849e981b2cd7a65f417fe",
		//		"amount": 5000,
		//		"items": [
		//	{
		//	"name": "Табуретка",
		//	"qty": 2,
		//	"sum": 2100,
		//	"code": "d21da1c47f3c45fca10a10c32518bdeb",
		//	"barcode": "3b2a558cc6e44e218cdce301d80a1779",
		//	"header": "Хідер",
		//	"footer": "Футер",
		//	"tax": [
		//	0
		//	],
		//	"uktzed": "uktzedcode"
		//	}
		//]
		//}
	case "status":
		var data GetStatusBody
		appErr := scope.Decode(argv.Invoice.Body, &data)
		if appErr != nil {
			return model.CallResponseError, appErr
		}
		appErr = data.Normalize()
		if appErr != nil {
			return model.CallResponseError, appErr
		}
		res, err := GetPaymentState(ctx, data, argv.Token)
		if err != nil {
			return model.CallResponseError, model.NewAppError("", "flow.monopay.monopay_handler.payment_status.error", nil, err.Error(), http.StatusInternalServerError)
		}
		return conn.Set(ctx, model.Variables{
			data.SetVar: string(res),
		})
		//	{
		//		"invoiceId": "p2_9ZgpZVsl3",
		//		"status": "created",
		//		"failureReason": "Неправильний CVV код",
		//		"amount": 4200,
		//		"ccy": 980,
		//		"finalAmount": 4200,
		//		"createdDate": "2019-08-24T14:15:22Z",
		//		"modifiedDate": "2019-08-24T14:15:22Z",
		//		"reference": "84d0070ee4e44667b31371d8f8813947",
		//		"cancelList": [
		//	{
		//	"status": "processing",
		//	"amount": 4200,
		//	"ccy": 980,
		//	"createdDate": "2019-08-24T14:15:22Z",
		//	"modifiedDate": "2019-08-24T14:15:22Z",
		//	"approvalCode": "662476",
		//	"rrn": "060189181768",
		//	"extRef": "635ace02599849e981b2cd7a65f417fe"
		//	}
		//	],
		//	"walletData": {
		//"cardToken": "67XZtXdR4NpKU3",
		//"walletId": "c1376a611e17b059aeaf96b73258da9c",
		//"status": "new"
		//}
		//}
	case "payment_info":
		var data GetSuccessfulPaymentInfoBody
		appErr := scope.Decode(argv.Invoice.Body, &data)
		if appErr != nil {
			return model.CallResponseError, appErr
		}
		res, err := GetSuccesfulPaymentInfo(ctx, data, argv.Token)
		if err != nil {
			return model.CallResponseError, model.NewAppError("", "flow.monopay.monopay_handler.cancel_payment.error", nil, err.Error(), http.StatusInternalServerError)
		}
		return conn.Set(ctx, model.Variables{
			data.SetVar: string(res),
		})
	default:
		return model.CallResponseError, model.NewAppError("", "flow.monopay.monopay_handler.check_args.error", nil, "unknown operation type", http.StatusBadRequest)

	}
}

func CreatePayment(ctx context.Context, body CreateBody, token string) ([]byte, error) {
	var (
		req *http.Request
	)
	s, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	// region CONSTRUCTING REQUEST
	req, err = http.NewRequest(http.MethodPost, createUrl, bytes.NewReader(s))
	req.Header.Set("X-Token", token)
	req.Header.Set("X-Cms", "Webitel")
	req.Header.Set("Content-Type", "application/json")
	req.WithContext(ctx)
	// endregion

	// region PEFORM
	httpResponse, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer httpResponse.Body.Close()
	// endregion
	return GetMonoRequestResults(httpResponse)

}

func CancelPayment(ctx context.Context, body CancelBody, token string) ([]byte, error) {
	var (
		req *http.Request
	)
	s, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	// region CONSTRUCTING REQUEST
	req, err = http.NewRequest(http.MethodPost, cancelUrl, bytes.NewReader(s))
	req.Header.Set("X-Token", token)
	req.WithContext(ctx)
	// endregion

	// region PEFORM
	httpResponse, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer httpResponse.Body.Close()
	// endregion

	return GetMonoRequestResults(httpResponse)
}

func InvalidatePayment(ctx context.Context, body InvalidateBody, token string) error {
	var (
		req     *http.Request
		monoErr MonoErr
	)
	s, err := json.Marshal(body)
	if err != nil {
		return err
	}
	// region CONSTRUCTING REQUEST
	req, err = http.NewRequest(http.MethodPost, invalidateUrl, bytes.NewReader(s))
	req.Header.Set("X-Token", token)
	req.WithContext(ctx)
	// endregion

	// region PEFORM
	httpResponse, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer httpResponse.Body.Close()
	httpBody, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		return err
	}
	err = json.Unmarshal(httpBody, &monoErr)
	if err != nil {
		return err
	}
	if monoErr.IsValid() {
		return FormatMonoError(monoErr)
	}
	// endregion

	return nil
}

func GetPaymentState(ctx context.Context, body GetStatusBody, token string) ([]byte, error) {
	var (
		req *http.Request
	)
	// region CONSTRUCTING REQUEST
	req, err := http.NewRequest(http.MethodGet, statusUrl, nil)
	req.Header.Set("X-Token", token)
	req.WithContext(ctx)
	values := req.URL.Query()
	values.Add("invoiceId", body.InvoiceId)
	req.URL.RawQuery = values.Encode()
	// endregion

	// region PEFORM
	httpResponse, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer httpResponse.Body.Close()
	// endregion

	return GetMonoRequestResults(httpResponse)
}

func GetSuccesfulPaymentInfo(ctx context.Context, body GetSuccessfulPaymentInfoBody, token string) ([]byte, error) {
	var (
		req *http.Request
	)
	s, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	// region CONSTRUCTING REQUEST
	req, err = http.NewRequest(http.MethodGet, successfulPaymentInfoUrl, bytes.NewReader(s))
	req.Header.Set("X-Token", token)
	req.WithContext(ctx)
	values := req.URL.Query()
	values.Add("invoiceId", body.InvoiceId)
	req.URL.RawQuery = values.Encode()

	// endregion

	// region PEFORM
	httpResponse, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer httpResponse.Body.Close()
	// endregion
	return GetMonoRequestResults(httpResponse)
}

func GetMonoRequestResults(httpResponse *http.Response) ([]byte, error) {
	var monoErr MonoErr
	httpBody, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(httpBody, &monoErr)
	if err != nil {
		return nil, err
	}
	if monoErr.ErrCode != "" {
		return nil, FormatMonoError(monoErr)
	}
	return httpBody, nil
}

// region TYPES

type MonopayArgs struct {
	Token   string `json:"token"`
	Invoice struct {
		Action string            `json:"action"`
		Body   map[string]string `json:"body"`
	} `json:"invoice"`
}

type CreateBody struct {
	// Amount of currency
	Amount int `json:"amount,omitempty"`
	// Currency code by ISO 4217
	Ccy int `json:"ccy,omitempty"`
	// Payment type
	PaymentType string `json:"paymentType,omitempty"`
	// Redirect Url
	RedirectUrl string `json:"redirectUrl,omitempty"`
	// Validity -- time that payment will be valid in secs
	ValidFor int `json:"validity,omitempty"`
	// Url that will process webhook events
	WebHookUrl string `json:"webHookUrl,omitempty"`
	// Set response to the variable
	SetVar string `json:"setVar"`
}

func (c *CreateBody) Normalize() *model.AppError {
	if c.ValidFor == 0 {
		c.ValidFor = defaultValidity
	}
	if c.SetVar == "" {
		return model.NewAppError("", "flow.monopay.create.check_arg.error", nil, "mono: setVar not set", http.StatusBadRequest)
	}
	switch c.PaymentType {
	case "debit", "hold":
	default:
		return model.NewAppError("", "flow.monopay.create.check_arg.error", nil, "mono: unknown payment type", http.StatusBadRequest)
	}
	return nil
}

func FormatMonoError(err MonoErr) error {
	return errors.New(fmt.Sprintf("mono: Code=%s, Error: %s", err.ErrCode, err.ErrText))
}

type GetStatusBody struct {
	InvoiceId string `json:"invoiceId,omitempty"`
	// Set response to the variable
	SetVar string `json:"setVar"`
}

func (c *GetStatusBody) Normalize() *model.AppError {
	if c.SetVar == "" {
		return model.NewAppError("", "flow.monopay.status.check_arg.error", nil, "mono: setVar not set", http.StatusBadRequest)
	}
	if c.InvoiceId == "" {
		return model.NewAppError("", "flow.monopay.status.check_arg.error", nil, "mono: invoice id empty", http.StatusBadRequest)
	}
	return nil
}

type InvalidateBody struct {
	InvoiceId string `json:"invoiceId,omitempty"`
}

func (c *InvalidateBody) Normalize() *model.AppError {
	if c.InvoiceId == "" {
		return model.NewAppError("", "flow.monopay.invalidate.check_arg.error", nil, "mono: invoice id empty", http.StatusBadRequest)
	}
	return nil
}

type CancelBody struct {
	// Additional
	InvoiceId string `json:"invoiceId,omitempty"`
	ExtRef    string `json:"extRef,omitempty"`
	Amount    int    `json:"amount,omitempty"`
	// Set response to the variable
	SetVar string `json:"setVar,omitempty"`
}

func (c *CancelBody) Normalize() *model.AppError {
	if c.InvoiceId == "" {
		return model.NewAppError("", "flow.monopay.cancel.check_arg.error", nil, "mono: invoice id empty", http.StatusBadRequest)
	}
	if c.SetVar == "" {
		return model.NewAppError("", "flow.monopay.cancel.check_arg.error", nil, "mono: setVar not set", http.StatusBadRequest)
	}
	return nil
}

type GetSuccessfulPaymentInfoBody struct {
	InvoiceId string `json:"invoiceId,omitempty"`
	// Set response to the variable
	SetVar string `json:"setVar,omitempty"`
}

func (c *GetSuccessfulPaymentInfoBody) Normalize() *model.AppError {
	if c.InvoiceId == "" {
		return model.NewAppError("", "flow.monopay.payment_info.check_arg.error", nil, "mono: invoice id empty", http.StatusBadRequest)
	}
	if c.SetVar == "" {
		return model.NewAppError("", "flow.monopay.payment_info.check_arg.error", nil, "mono: setVar not set", http.StatusBadRequest)
	}
	return nil
}

type MonoErr struct {
	ErrCode string `json:"errCode,omitempty"`
	ErrText string `json:"errText,omitempty"`
}

func (e MonoErr) IsValid() bool {
	return e.ErrCode != "" && e.ErrText != ""
}

// endregion
