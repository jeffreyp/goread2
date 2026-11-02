package services

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jeffreyp/goread2/internal/database"
	"github.com/stripe/stripe-go/v78"
)

// stripeResponse is a canned reply from fakeStripe.
type stripeResponse struct {
	status int
	body   string
}

const (
	stripeMissingBody = `{"error":{"type":"invalid_request_error","code":"resource_missing","message":"No such object"}}`
	// Stripe answers 403 to a restricted key that lacks a permission. The SDK
	// does not retry 4xx responses, so the test sees the error at once.
	stripeForbiddenBody = `{"error":{"type":"invalid_request_error","message":"The provided key does not have the required permissions"}}`
)

var (
	stripeMissing   = stripeResponse{http.StatusNotFound, stripeMissingBody}
	stripeForbidden = stripeResponse{http.StatusForbidden, stripeForbiddenBody}
)

// stripeOK wraps a JSON body in a 200 response.
func stripeOK(body string) stripeResponse { return stripeResponse{http.StatusOK, body} }

// stripeCustomerSearch builds a search result holding the given customer
// objects.
func stripeCustomerSearch(customers ...string) stripeResponse {
	return stripeOK(`{"object":"search_result","url":"/v1/customers/search","has_more":false,"data":[` +
		strings.Join(customers, ",") + `]}`)
}

// stripeCustomer builds a customer object. An empty userID omits the
// user_id metadata.
func stripeCustomer(id, userID string) string {
	metadata := "{}"
	if userID != "" {
		metadata = fmt.Sprintf(`{"user_id":%q}`, userID)
	}
	return fmt.Sprintf(`{"id":%q,"object":"customer","metadata":%s}`, id, metadata)
}

// fakeStripe is a Stripe API server for tests. It answers each request from
// responses, keyed by "METHOD /path", and records every request it receives.
// A DELETE /v1/customers/{id} with no matching key succeeds. Any other
// unexpected request fails the test.
type fakeStripe struct {
	t         *testing.T
	responses map[string]stripeResponse

	mu            sync.Mutex
	requests      []string
	searchQueries []string
	deleted       []string
}

// newFakeStripe points the Stripe SDK at a fakeStripe for the rest of the
// test.
func newFakeStripe(t *testing.T, responses map[string]stripeResponse) *fakeStripe {
	t.Helper()
	f := &fakeStripe{t: t, responses: responses}
	srv := httptest.NewServer(http.HandlerFunc(f.serve))
	restoreBackend := setStripeTestBackend(srv)
	previousKey := stripe.Key
	stripe.Key = "sk_test_fake"
	t.Cleanup(func() {
		stripe.Key = previousKey
		restoreBackend()
		srv.Close()
	})
	return f
}

func (f *fakeStripe) serve(w http.ResponseWriter, r *http.Request) {
	key := r.Method + " " + r.URL.Path
	f.mu.Lock()
	f.requests = append(f.requests, key)
	if r.URL.Path == "/v1/customers/search" {
		f.searchQueries = append(f.searchQueries, r.URL.Query().Get("query"))
	}
	isDelete := r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/v1/customers/")
	if isDelete {
		f.deleted = append(f.deleted, strings.TrimPrefix(r.URL.Path, "/v1/customers/"))
	}
	f.mu.Unlock()

	resp, ok := f.responses[key]
	if !ok && isDelete {
		id := strings.TrimPrefix(r.URL.Path, "/v1/customers/")
		resp, ok = stripeOK(fmt.Sprintf(`{"id":%q,"object":"customer","deleted":true}`, id)), true
	}
	if !ok {
		f.t.Errorf("unexpected Stripe request %s", key)
		resp = stripeResponse{http.StatusBadRequest, `{"error":{"type":"invalid_request_error","message":"unexpected request"}}`}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.status)
	_, _ = w.Write([]byte(resp.body))
}

// deletedCustomers returns the IDs of deleted customers, sorted, since
// deletion order follows map iteration.
func (f *fakeStripe) deletedCustomers() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	deleted := append([]string{}, f.deleted...)
	sort.Strings(deleted)
	return deleted
}

func TestPaymentService_DeleteCustomerForUser(t *testing.T) {
	const subPath = "GET /v1/subscriptions/sub_1"
	const searchPath = "GET /v1/customers/search"
	subscribed := &database.User{ID: 7, Email: "o'brien@example.com", SubscriptionID: "sub_1"}
	unsubscribed := &database.User{ID: 7, Email: "o'brien@example.com"}

	// The search returns a customer tagged for this user, one with no
	// user_id metadata (possibly created by hand in the Dashboard), and one
	// tagged for another user who shares the address.
	searchResults := stripeCustomerSearch(
		stripeCustomer("cus_tagged", "7"),
		stripeCustomer("cus_untagged", ""),
		stripeCustomer("cus_other_user", "8"),
	)

	tests := []struct {
		name         string
		user         *database.User
		responses    map[string]stripeResponse
		wantErr      string
		wantRequests []string // in order, excluding deletions
		wantDeleted  []string
	}{
		{
			name: "deletes the subscription's customer and email matches tagged with the user",
			user: subscribed,
			responses: map[string]stripeResponse{
				subPath:    stripeOK(`{"id":"sub_1","object":"subscription","customer":"cus_sub"}`),
				searchPath: searchResults,
			},
			wantRequests: []string{subPath, searchPath},
			wantDeleted:  []string{"cus_sub", "cus_tagged"},
		},
		{
			name: "a subscription customer also found by email is deleted once",
			user: subscribed,
			responses: map[string]stripeResponse{
				subPath:    stripeOK(`{"id":"sub_1","object":"subscription","customer":"cus_tagged"}`),
				searchPath: searchResults,
			},
			wantRequests: []string{subPath, searchPath},
			wantDeleted:  []string{"cus_tagged"},
		},
		{
			name:         "a user without a subscription is found by email only",
			user:         unsubscribed,
			responses:    map[string]stripeResponse{searchPath: searchResults},
			wantRequests: []string{searchPath},
			wantDeleted:  []string{"cus_tagged"},
		},
		{
			name: "an already deleted subscription is ignored",
			user: subscribed,
			responses: map[string]stripeResponse{
				subPath:    stripeMissing,
				searchPath: searchResults,
			},
			wantRequests: []string{subPath, searchPath},
			wantDeleted:  []string{"cus_tagged"},
		},
		{
			name: "a subscription lookup failure aborts before any deletion",
			user: subscribed,
			responses: map[string]stripeResponse{
				subPath: stripeForbidden,
			},
			wantErr:      "failed to get subscription",
			wantRequests: []string{subPath},
		},
		{
			name: "a search failure aborts before any deletion",
			user: subscribed,
			responses: map[string]stripeResponse{
				subPath:    stripeOK(`{"id":"sub_1","object":"subscription","customer":"cus_sub"}`),
				searchPath: stripeForbidden,
			},
			wantErr:      "failed to search for customers",
			wantRequests: []string{subPath, searchPath},
		},
		{
			name: "an already deleted customer is ignored",
			user: unsubscribed,
			responses: map[string]stripeResponse{
				searchPath:                        searchResults,
				"DELETE /v1/customers/cus_tagged": stripeMissing,
			},
			wantRequests: []string{searchPath},
			wantDeleted:  []string{"cus_tagged"},
		},
		{
			name: "any other deletion failure is returned",
			user: unsubscribed,
			responses: map[string]stripeResponse{
				searchPath:                        searchResults,
				"DELETE /v1/customers/cus_tagged": stripeForbidden,
			},
			wantErr:      "failed to delete customer",
			wantRequests: []string{searchPath},
			wantDeleted:  []string{"cus_tagged"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := newFakeStripe(t, tt.responses)
			ps := newTestPaymentService(&mockDBPayment{}, "pk_test", "sk_test_fake", "whsec_test", "price_test")

			err := ps.DeleteCustomerForUser(tt.user)

			if tt.wantErr == "" && err != nil {
				t.Fatalf("DeleteCustomerForUser() error = %v, want nil", err)
			}
			if tt.wantErr != "" {
				var stripeErr *stripe.Error
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) || !errors.As(err, &stripeErr) {
					t.Fatalf("DeleteCustomerForUser() error = %v, want a wrapped *stripe.Error containing %q", err, tt.wantErr)
				}
			}

			var requests []string
			for _, r := range fake.requests {
				if !strings.HasPrefix(r, "DELETE ") {
					requests = append(requests, r)
				}
			}
			if !reflect.DeepEqual(requests, tt.wantRequests) {
				t.Errorf("requests = %v, want %v", requests, tt.wantRequests)
			}
			if want := tt.wantDeleted; !reflect.DeepEqual(fake.deletedCustomers(), append([]string{}, want...)) {
				t.Errorf("deleted customers = %v, want %v", fake.deletedCustomers(), want)
			}
			for _, q := range fake.searchQueries {
				if want := `email:'o\'brien@example.com'`; q != want {
					t.Errorf("customer search query = %q, want %q", q, want)
				}
			}
		})
	}
}

func TestIsStripeResourceMissing(t *testing.T) {
	missing := &stripe.Error{Code: stripe.ErrorCodeResourceMissing}
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"resource_missing", missing, true},
		{"wrapped resource_missing", fmt.Errorf("failed: %w", missing), true},
		{"another Stripe error code", &stripe.Error{Code: stripe.ErrorCodeParameterMissing}, false},
		{"a Stripe error without a code", &stripe.Error{HTTPStatusCode: http.StatusForbidden}, false},
		{"a non-Stripe error", errors.New("resource_missing"), false},
		{"nil", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isStripeResourceMissing(tt.err); got != tt.want {
				t.Errorf("isStripeResourceMissing(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

// mockDBSubscriptionUpdate serves the two Database calls
// HandleSubscriptionUpdate makes. Any other call panics.
type mockDBSubscriptionUpdate struct {
	database.Database
	getErr    error
	updateErr error
	updates   []subscriptionUpdate
}

type subscriptionUpdate struct {
	userID          int
	status          string
	subscriptionID  string
	lastPaymentDate time.Time
	nextBillingDate time.Time
}

func (m *mockDBSubscriptionUpdate) GetUserByID(userID int) (*database.User, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return &database.User{ID: userID}, nil
}

func (m *mockDBSubscriptionUpdate) UpdateUserSubscription(userID int, status, subscriptionID string, lastPaymentDate, nextBillingDate time.Time) error {
	m.updates = append(m.updates, subscriptionUpdate{userID, status, subscriptionID, lastPaymentDate, nextBillingDate})
	return m.updateErr
}

func TestPaymentService_HandleSubscriptionUpdate(t *testing.T) {
	const subPath = "GET /v1/subscriptions/sub_1"
	periodStart, periodEnd := int64(1790000000), int64(1792592000)
	subscription := func(status, metadata string) stripeResponse {
		return stripeOK(fmt.Sprintf(`{"id":"sub_1","object":"subscription","status":%q,"metadata":%s,`+
			`"current_period_start":%d,"current_period_end":%d}`, status, metadata, periodStart, periodEnd))
	}
	dbErr := errors.New("datastore unavailable")

	tests := []struct {
		name        string
		response    stripeResponse
		getErr      error
		updateErr   error
		wantErr     string
		wantUpdates []subscriptionUpdate
	}{
		{
			name:     "an active subscription records the billing period",
			response: subscription("active", `{"user_id":"7"}`),
			wantUpdates: []subscriptionUpdate{{
				7, "active", "sub_1", time.Unix(periodStart, 0), time.Unix(periodEnd, 0),
			}},
		},
		{
			name:        "a past_due subscription is recorded without dates",
			response:    subscription("past_due", `{"user_id":"7"}`),
			wantUpdates: []subscriptionUpdate{{7, "past_due", "sub_1", time.Time{}, time.Time{}}},
		},
		{
			name:        "an unrecognised status is recorded as cancelled",
			response:    subscription("incomplete_expired", `{"user_id":"7"}`),
			wantUpdates: []subscriptionUpdate{{7, "cancelled", "sub_1", time.Time{}, time.Time{}}},
		},
		{
			name:     "a deleted user is not an error and is not updated",
			response: subscription("canceled", `{"user_id":"7"}`),
			getErr:   database.ErrUserNotFound,
		},
		{
			name:        "a user deleted between the check and the update is not an error",
			response:    subscription("canceled", `{"user_id":"7"}`),
			updateErr:   database.ErrUserNotFound,
			wantUpdates: []subscriptionUpdate{{7, "cancelled", "sub_1", time.Time{}, time.Time{}}},
		},
		{
			name:        "any other update failure is returned",
			response:    subscription("active", `{"user_id":"7"}`),
			updateErr:   dbErr,
			wantErr:     "failed to update user subscription",
			wantUpdates: []subscriptionUpdate{{7, "active", "sub_1", time.Unix(periodStart, 0), time.Unix(periodEnd, 0)}},
		},
		{
			name:     "a subscription without user_id metadata is an error",
			response: subscription("active", `{}`),
			wantErr:  "user_id not found",
		},
		{
			name:     "a non-numeric user_id is an error",
			response: subscription("active", `{"user_id":"abc"}`),
			wantErr:  "invalid user_id",
		},
		{
			name:     "a subscription lookup failure is returned",
			response: stripeForbidden,
			wantErr:  "failed to get subscription",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			newFakeStripe(t, map[string]stripeResponse{subPath: tt.response})
			db := &mockDBSubscriptionUpdate{getErr: tt.getErr, updateErr: tt.updateErr}
			ps := &PaymentService{db: db, subscriptionService: NewSubscriptionService(db)}

			err := ps.HandleSubscriptionUpdate("sub_1")

			if tt.wantErr == "" && err != nil {
				t.Fatalf("HandleSubscriptionUpdate() error = %v, want nil", err)
			}
			if tt.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErr)) {
				t.Fatalf("HandleSubscriptionUpdate() error = %v, want error containing %q", err, tt.wantErr)
			}
			if tt.updateErr == dbErr && !errors.Is(err, dbErr) {
				t.Errorf("HandleSubscriptionUpdate() error = %v, want it to wrap %v", err, dbErr)
			}
			if !reflect.DeepEqual(db.updates, tt.wantUpdates) {
				t.Errorf("updates = %+v, want %+v", db.updates, tt.wantUpdates)
			}
		})
	}
}
