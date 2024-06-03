package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type PalPayClient struct {
	HttpClient http.RoundTripper
}

type ChargeRequest struct {
	CreditCardNumber string
	CVV2             string
	Amount           int
	IdempotencyKey   string
}

func (p *PalPayClient) Charge(req ChargeRequest) error {
	payload, err := json.Marshal(map[string]any{
		"credit_card_number": req.CreditCardNumber,
		"cvv2":               req.CVV2,
		"amount":             req.Amount,
	})
	if err != nil {
		return err
	}

	httpReq, err := http.NewRequest("POST", "https://palpay.io/charge", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	httpReq.Header.Set("Idempotency-Key", req.IdempotencyKey)
	resp, err := p.HttpClient.RoundTrip(httpReq)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}
