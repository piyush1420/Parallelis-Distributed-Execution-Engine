package model

// JobType represents the types of jobs in an e-commerce order processing system.
//
// Note: Inventory updates are handled synchronously during payment processing
// to ensure atomicity and prevent double-decrement issues.
type JobType string

const (
	// TypePaymentProcess processes payment through payment gateway.
	//
	// Simulated processing time: 5 seconds
	//
	// This includes atomic inventory check and decrement:
	// 1. Begin database transaction
	// 2. Check if product is in stock (SELECT ... FOR UPDATE)
	// 3. Call payment gateway API to charge card
	// 4. If payment succeeds, decrement inventory
	// 5. Commit transaction
	//
	// Payload format: "order_12345|customer@email.com|$99.99|product_SKU123|qty_2"
	//
	// Retry scenarios (transient failures):
	// - Payment gateway timeout (network issue)
	// - Gateway temporarily overloaded (5xx errors)
	// - Transient API error (503 Service Unavailable)
	// - Rate limit exceeded (429 Too Many Requests)
	// - Database connection timeout (before payment charged)
	//
	// Non-retriable scenarios:
	// - Payment succeeded but inventory update failed → Manual intervention required
	// - Card declined (insufficient funds) → Permanent failure
	// - Product out of stock → Permanent failure
	TypePaymentProcess JobType = "PAYMENT_PROCESS"

	// TypeEmailConfirmation sends order confirmation email to customer.
	//
	// Simulated processing time: 1 second
	// Real-world operation: Send receipt via email service (SendGrid, AWS SES, Mailgun)
	//
	// Payload format: "order_12345|customer@email.com|$99.99|tracking_url"
	//
	// Retry scenarios (transient failures):
	// - SMTP server busy (too many connections)
	// - Email provider rate limit hit (temporary throttle)
	// - Network blip (connection reset)
	// - Temporary DNS resolution failure
	//
	// Non-retriable scenarios:
	// - Invalid email address (bounce)
	// - Domain does not exist
	//
	// Why retry: Customers expect confirmation emails. Legal requirement in some regions.
	// Better UX and reduces customer support inquiries.
	//
	// Note: This job is only created AFTER payment succeeds. If payment fails,
	// no email confirmation job is created.
	TypeEmailConfirmation JobType = "EMAIL_CONFIRMATION"
)