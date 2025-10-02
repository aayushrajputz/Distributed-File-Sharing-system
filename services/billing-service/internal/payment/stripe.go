package payment

import (
	"encoding/json"
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/checkout/session"
	"github.com/stripe/stripe-go/v76/webhook"
	"github.com/yourusername/distributed-file-sharing-platform/services/billing-service/internal/models"
)

type StripeService struct {
	secretKey     string
	webhookSecret string
	successURL    string
	cancelURL     string
}

func NewStripeService(secretKey, webhookSecret, successURL, cancelURL string) *StripeService {
	stripe.Key = secretKey
	return &StripeService{
		secretKey:     secretKey,
		webhookSecret: webhookSecret,
		successURL:    successURL,
		cancelURL:     cancelURL,
	}
}

// CreateCheckoutSession creates a Stripe checkout session for a subscription
func (s *StripeService) CreateCheckoutSession(plan *models.Plan, userID, subscriptionID string) (*stripe.CheckoutSession, error) {
	params := &stripe.CheckoutSessionParams{
		PaymentMethodTypes: stripe.StringSlice([]string{"card"}),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
					Currency: stripe.String("usd"),
					ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
						Name:        stripe.String(plan.Name + " Plan"),
						Description: stripe.String(plan.Description),
					},
					UnitAmount: stripe.Int64(int64(plan.PricePerMonth * 100)), // Convert to cents
				},
				Quantity: stripe.Int64(1),
			},
		},
		Mode:              stripe.String(string(stripe.CheckoutSessionModePayment)),
		SuccessURL:        stripe.String(s.successURL + "?session_id={CHECKOUT_SESSION_ID}"),
		CancelURL:         stripe.String(s.cancelURL),
		ClientReferenceID: stripe.String(subscriptionID),
		Metadata: map[string]string{
			"user_id":         userID,
			"subscription_id": subscriptionID,
			"plan_id":         plan.ID.Hex(),
			"plan_name":       plan.Name,
		},
	}

	sess, err := session.New(params)
	if err != nil {
		return nil, fmt.Errorf("failed to create checkout session: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"session_id":      sess.ID,
		"user_id":         userID,
		"subscription_id": subscriptionID,
		"plan":            plan.Name,
	}).Info("Stripe checkout session created")

	return sess, nil
}

// VerifyWebhookSignature verifies the Stripe webhook signature
func (s *StripeService) VerifyWebhookSignature(payload []byte, signature string) (stripe.Event, error) {
	event, err := webhook.ConstructEvent(payload, signature, s.webhookSecret)
	if err != nil {
		return stripe.Event{}, fmt.Errorf("failed to verify webhook signature: %w", err)
	}
	return event, nil
}

// ParseCheckoutSessionCompleted parses a checkout.session.completed event
func (s *StripeService) ParseCheckoutSessionCompleted(event stripe.Event) (*CheckoutSessionData, error) {
	var sess stripe.CheckoutSession
	if err := json.Unmarshal(event.Data.Raw, &sess); err != nil {
		return nil, fmt.Errorf("failed to parse checkout session: %w", err)
	}

	data := &CheckoutSessionData{
		SessionID:      sess.ID,
		PaymentStatus:  string(sess.PaymentStatus),
		CustomerEmail:  sess.CustomerDetails.Email,
		AmountTotal:    sess.AmountTotal,
		Currency:       string(sess.Currency),
		UserID:         sess.Metadata["user_id"],
		SubscriptionID: sess.Metadata["subscription_id"],
		PlanID:         sess.Metadata["plan_id"],
		PlanName:       sess.Metadata["plan_name"],
	}

	// Get payment intent ID as transaction ID
	if sess.PaymentIntent != nil {
		data.TransactionID = sess.PaymentIntent.ID
	}

	return data, nil
}

// CheckoutSessionData represents parsed checkout session data
type CheckoutSessionData struct {
	SessionID      string
	TransactionID  string
	PaymentStatus  string
	CustomerEmail  string
	AmountTotal    int64
	Currency       string
	UserID         string
	SubscriptionID string
	PlanID         string
	PlanName       string
}

// GetSessionDetails retrieves details of a checkout session
func (s *StripeService) GetSessionDetails(sessionID string) (*stripe.CheckoutSession, error) {
	sess, err := session.Get(sessionID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get session details: %w", err)
	}
	return sess, nil
}

// CreateRecurringSubscription creates a recurring subscription (for future use)
func (s *StripeService) CreateRecurringSubscription(plan *models.Plan, customerID string) (*stripe.Subscription, error) {
	// This would be used for recurring billing
	// For now, we're doing one-time payments
	// Implementation can be added later if needed
	return nil, fmt.Errorf("recurring subscriptions not implemented yet")
}

// RefundPayment refunds a payment
func (s *StripeService) RefundPayment(paymentIntentID string, amount int64) error {
	// Implementation for refunds
	// Can be added when needed
	return fmt.Errorf("refunds not implemented yet")
}

// HandleWebhookEvent handles different types of Stripe webhook events
func (s *StripeService) HandleWebhookEvent(event stripe.Event) (*WebhookResult, error) {
	result := &WebhookResult{
		EventType: string(event.Type),
		Processed: false,
	}

	switch event.Type {
	case "checkout.session.completed":
		data, err := s.ParseCheckoutSessionCompleted(event)
		if err != nil {
			return nil, err
		}
		result.Data = data
		result.Processed = true
		logrus.WithFields(logrus.Fields{
			"event_type":      event.Type,
			"session_id":      data.SessionID,
			"subscription_id": data.SubscriptionID,
		}).Info("Checkout session completed")

	case "checkout.session.expired":
		var sess stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &sess); err != nil {
			return nil, fmt.Errorf("failed to parse expired session: %w", err)
		}
		result.Data = &CheckoutSessionData{
			SessionID:      sess.ID,
			SubscriptionID: sess.Metadata["subscription_id"],
			UserID:         sess.Metadata["user_id"],
		}
		result.Processed = true
		logrus.WithFields(logrus.Fields{
			"event_type": event.Type,
			"session_id": sess.ID,
		}).Info("Checkout session expired")

	case "payment_intent.succeeded":
		logrus.WithField("event_type", event.Type).Info("Payment intent succeeded")
		result.Processed = true

	case "payment_intent.payment_failed":
		logrus.WithField("event_type", event.Type).Warn("Payment intent failed")
		result.Processed = true

	default:
		logrus.WithField("event_type", event.Type).Debug("Unhandled webhook event")
	}

	return result, nil
}

// WebhookResult represents the result of processing a webhook
type WebhookResult struct {
	EventType string
	Processed bool
	Data      interface{}
}
