package dto

import (
	"fmt"

	"distributed-job-processor/model"
)

// JobRequest is the request DTO for creating a new job.
//
// Example payloads:
// - PAYMENT_PROCESS: "order_12345|customer@email.com|$99.99|card_tok_xyz"
// - INVENTORY_UPDATE: "product_SKU123|quantity_5|warehouse_US_EAST"
// - EMAIL_CONFIRMATION: "order_12345|customer@email.com|receipt_url"
type JobRequest struct {
	Type    model.JobType `json:"type" binding:"required"`
	Payload string        `json:"payload" binding:"required"`
}

// ForPaymentProcess is a factory method to create a payment processing job request.
func ForPaymentProcess(orderID string, customerEmail string, amount string) JobRequest {
	payload := fmt.Sprintf("%s|%s|%s", orderID, customerEmail, amount)
	return JobRequest{
		Type:    model.TypePaymentProcess,
		Payload: payload,
	}
}

// ForEmailConfirmation is a factory method to create an email confirmation job request.
func ForEmailConfirmation(orderID string, customerEmail string, receiptURL string) JobRequest {
	payload := fmt.Sprintf("%s|%s|%s", orderID, customerEmail, receiptURL)
	return JobRequest{
		Type:    model.TypeEmailConfirmation,
		Payload: payload,
	}
}