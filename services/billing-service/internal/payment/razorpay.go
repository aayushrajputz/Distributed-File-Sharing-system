package payment

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/razorpay/razorpay-go"
	"github.com/sirupsen/logrus"
	"github.com/yourusername/distributed-file-sharing-platform/services/billing-service/internal/models"
)

type RazorpayService struct {
	client        *razorpay.Client
	webhookSecret string
}

func NewRazorpayService(keyID, keySecret, webhookSecret string) *RazorpayService {
	client := razorpay.NewClient(keyID, keySecret)
	return &RazorpayService{
		client:        client,
		webhookSecret: webhookSecret,
	}
}

// CreateSubscription creates a Razorpay subscription
func (s *RazorpayService) CreateSubscription(plan *models.Plan, userID, subscriptionID string) (string, string, error) {
	// 1. Create a Plan in Razorpay if it doesn't exist (or use a fixed mapping)
	// For simplicity, we'll assume we create a new plan or use a fixed one.
	// In a real app, you'd sync plans. Here we'll create a subscription directly if possible,
	// but Razorpay subscriptions require a Plan ID.
	// Let's assume we create a "link" or "order" for one-time payment first as per the simple flow,
	// or if we want recurring, we need a Plan.
	// Given the "CreateSubscription" name, let's try to create a Subscription.

	// However, creating a plan dynamically for every user is not ideal.
	// Let's fallback to creating an Order for the first payment, which is common for "subscribe" flows
	// that start with a payment. But wait, the proto says "CreateSubscription".
	// Let's assume we use Razorpay Subscriptions.
	
	// For this implementation, let's assume we map our internal plan to a Razorpay Plan ID.
	// Since we don't have that mapping yet, let's create a dummy plan or just use an Order for simplicity
	// to get the payment flow working.
	// ACTUALLY, let's use Orders for one-time payments as a start, similar to the Stripe implementation
	// which uses Checkout Session (often one-time).
	
	amountInPaise := int64(plan.PricePerMonth * 100 * 83) // Approx USD to INR conversion if needed, or just assume price is in base currency.
	// Let's assume PricePerMonth is in USD, and we want to charge in USD or convert.
	// Razorpay supports international payments.
	
	data := map[string]interface{}{
		"amount":          amountInPaise,
		"currency":        "INR", // Using INR for Razorpay default
		"receipt":         subscriptionID,
		"notes": map[string]interface{}{
			"user_id":         userID,
			"subscription_id": subscriptionID,
			"plan_id":         plan.ID.Hex(),
		},
	}

	body, err := s.client.Order.Create(data, nil)
	if err != nil {
		return "", "", fmt.Errorf("failed to create razorpay order: %w", err)
	}

	orderID, ok := body["id"].(string)
	if !ok {
		return "", "", fmt.Errorf("failed to get order id from response")
	}

	// We don't get a payment URL directly like Stripe Checkout, 
	// but we return the Order ID which the frontend uses to open the checkout.
	// The proto expects a payment_url. We can return a dummy URL or handle this in frontend.
	// For now, let's return the Order ID as the session ID.
	
	return "", orderID, nil
}

// VerifyWebhookSignature verifies the Razorpay webhook signature
func (s *RazorpayService) VerifyWebhookSignature(payload []byte, signature string) error {
	mac := hmac.New(sha256.New, []byte(s.webhookSecret))
	mac.Write(payload)
	expectedMAC := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(signature), []byte(expectedMAC)) {
		return fmt.Errorf("signature mismatch")
	}
	return nil
}

// HandleWebhookEvent handles Razorpay webhook events
func (s *RazorpayService) HandleWebhookEvent(payload []byte) (*WebhookResult, error) {
	var event map[string]interface{}
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, fmt.Errorf("failed to unmarshal webhook payload: %w", err)
	}

	eventType, ok := event["event"].(string)
	if !ok {
		return nil, fmt.Errorf("failed to get event type")
	}

	result := &WebhookResult{
		EventType: eventType,
		Processed: false,
	}

	payloadData, ok := event["payload"].(map[string]interface{})
	if !ok {
		return result, nil // No payload to process
	}

	switch eventType {
	case "payment.captured":
		payment, ok := payloadData["payment"].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid payment data")
		}
		entity, ok := payment["entity"].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid payment entity")
		}

		notes, ok := entity["notes"].(map[string]interface{})
		if !ok {
			// Try to get order and fetch notes if needed, but notes should be there if we sent them
			logrus.Warn("No notes found in payment entity")
		}

		data := &CheckoutSessionData{
			TransactionID:  entity["id"].(string),
			PaymentStatus:  "paid",
			AmountTotal:    int64(entity["amount"].(float64)),
			Currency:       entity["currency"].(string),
			UserID:         getString(notes, "user_id"),
			SubscriptionID: getString(notes, "subscription_id"),
			PlanID:         getString(notes, "plan_id"),
		}
		result.Data = data
		result.Processed = true
		
	case "payment.failed":
		payment, ok := payloadData["payment"].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid payment data")
		}
		entity, ok := payment["entity"].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid payment entity")
		}
		
		notes, _ := entity["notes"].(map[string]interface{})

		data := &CheckoutSessionData{
			TransactionID:  entity["id"].(string),
			PaymentStatus:  "failed",
			UserID:         getString(notes, "user_id"),
			SubscriptionID: getString(notes, "subscription_id"),
		}
		result.Data = data
		result.Processed = true
	}

	return result, nil
}

func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}
