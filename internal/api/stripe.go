package api

import (
	"encoding/json"
	"fmt"
	"github.com/flyflow-devs/flyflow/internal/slack"
	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/customer"
	"github.com/stripe/stripe-go/v72/paymentmethod"
	"net/http"
)

type PaymentMethodResponse struct {
	ID           string `json:"id"`
	Type         string `json:"type"`
	Last4        string `json:"last4"`
	Brand        string `json:"brand"`
	ExpiryMonth  uint64  `json:"expiry_month"`
	ExpiryYear   uint64  `json:"expiry_year"`
	IsDefault    bool   `json:"is_default"`
}


func (a *API) SetPaymentMethod(w http.ResponseWriter, r *http.Request) {
	// Validate the API key
	user, err := a.ValidateAuth(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	slack.PostMessage(fmt.Sprintf("%s set primary payment method", user.Email))

	// Parse the request body
	var req struct {
		PaymentMethodID string `json:"payment_method_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Set up Stripe API key
	stripe.Key = a.Cfg.StripeSecretKey

	var stripeCustomer *stripe.Customer

	// Check if the user already has a Stripe customer ID
	if user.StripeCustomerID == "" {
		// Create a new Stripe customer
		params := &stripe.CustomerParams{
			Email: stripe.String(user.Email),
		}
		stripeCustomer, err = customer.New(params)
		if err != nil {
			http.Error(w, "Failed to create Stripe customer", http.StatusInternalServerError)
			return
		}

		// Update user with Stripe customer ID
		user.StripeCustomerID = stripeCustomer.ID
		if err := a.DB.Save(user).Error; err != nil {
			http.Error(w, "Failed to update user", http.StatusInternalServerError)
			return
		}
	} else {
		// Retrieve existing Stripe customer
		stripeCustomer, err = customer.Get(user.StripeCustomerID, nil)
		if err != nil {
			http.Error(w, "Failed to retrieve Stripe customer", http.StatusInternalServerError)
			return
		}
	}

	// Attach the payment method to the customer
	_, err = paymentmethod.Attach(
		req.PaymentMethodID,
		&stripe.PaymentMethodAttachParams{
			Customer: stripe.String(stripeCustomer.ID),
		},
	)
	if err != nil {
		http.Error(w, "Failed to attach payment method", http.StatusInternalServerError)
		return
	}

	// Update the customer's default payment method
	_, err = customer.Update(
		stripeCustomer.ID,
		&stripe.CustomerParams{
			InvoiceSettings: &stripe.CustomerInvoiceSettingsParams{
				DefaultPaymentMethod: stripe.String(req.PaymentMethodID),
			},
		},
	)
	if err != nil {
		http.Error(w, "Failed to update customer's default payment method", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Payment method set successfully"})
}

func (a *API) GetPaymentMethods(w http.ResponseWriter, r *http.Request) {
	// Validate the API key
	user, err := a.ValidateAuth(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Set up Stripe API key
	stripe.Key = a.Cfg.StripeSecretKey

	// Check if the user has a Stripe customer ID
	if user.StripeCustomerID == "" {
		// If not, return an empty list
		json.NewEncoder(w).Encode([]PaymentMethodResponse{})
		return
	}

	// List payment methods for the customer
	params := &stripe.PaymentMethodListParams{
		Customer: stripe.String(user.StripeCustomerID),
		Type:     stripe.String(string(stripe.PaymentMethodTypeCard)),
	}
	i := paymentmethod.List(params)

	var paymentMethods []PaymentMethodResponse

	for i.Next() {
		pm := i.PaymentMethod()

		// Get the default payment method for the customer
		paymentMethod := PaymentMethodResponse{
			ID:          pm.ID,
			Type:        string(pm.Type),
			Last4:       pm.Card.Last4,
			Brand:       string(pm.Card.Brand),
			ExpiryMonth: pm.Card.ExpMonth,
			ExpiryYear:  pm.Card.ExpYear,
		}

		paymentMethods = append(paymentMethods, paymentMethod)
	}

	if err := i.Err(); err != nil {
		http.Error(w, "Failed to list payment methods", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(paymentMethods)
}