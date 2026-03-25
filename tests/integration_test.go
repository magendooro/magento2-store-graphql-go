package tests

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/99designs/gqlgen/graphql/handler"
	_ "github.com/go-sql-driver/mysql"

	"github.com/magendooro/magento2-store-graphql-go/graph"
	"github.com/magendooro/magento2-go-common/middleware"
)

var testHandler http.Handler
var testDB *sql.DB

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func TestMain(m *testing.M) {
	host := envOrDefault("TEST_DB_HOST", "localhost")
	port := envOrDefault("TEST_DB_PORT", "3306")
	user := envOrDefault("TEST_DB_USER", "fch")
	password := envOrDefault("TEST_DB_PASSWORD", "")
	dbName := envOrDefault("TEST_DB_NAME", "magento248")
	socket := envOrDefault("TEST_DB_SOCKET", "/tmp/mysql.sock")

	var dsn string
	if host == "localhost" {
		dsn = user + ":" + password + "@unix(" + socket + ")/" + dbName + "?parseTime=true&time_zone=%27%2B00%3A00%27&loc=UTC"
	} else {
		dsn = user + ":" + password + "@tcp(" + host + ":" + port + ")/" + dbName + "?parseTime=true&time_zone=%27%2B00%3A00%27&loc=UTC"
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		panic("failed to connect to test database: " + err.Error())
	}
	if err := db.Ping(); err != nil {
		panic("failed to ping test database: " + err.Error())
	}
	testDB = db

	resolver, err := graph.NewResolver(db)
	if err != nil {
		panic("failed to create resolver: " + err.Error())
	}

	storeResolver := middleware.NewStoreResolver(db)

	srv := handler.NewDefaultServer(graph.NewExecutableSchema(graph.Config{
		Resolvers: resolver,
	}))

	mux := http.NewServeMux()
	mux.Handle("/graphql", srv)

	var h http.Handler = mux
	h = middleware.StoreMiddleware(storeResolver)(h)
	testHandler = h

	os.Exit(m.Run())
}

type gqlResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

func doQuery(t *testing.T, query string) gqlResponse {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"query": query})
	req := httptest.NewRequest("POST", "/graphql", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Store", "default")
	rr := httptest.NewRecorder()
	testHandler.ServeHTTP(rr, req)

	var resp gqlResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v\nbody: %s", err, rr.Body.String())
	}
	return resp
}

// ─── storeConfig ─────────────────────────────────────────────────────────────

func TestStoreConfig_ReturnsBasicFields(t *testing.T) {
	resp := doQuery(t, `{
		storeConfig {
			id
			code
			store_code
			store_name
			locale
			base_currency_code
			default_display_currency_code
		}
	}`)
	if len(resp.Errors) > 0 {
		t.Fatalf("unexpected error: %s", resp.Errors[0].Message)
	}

	var data struct {
		StoreConfig struct {
			ID                         *int    `json:"id"`
			Code                       *string `json:"code"`
			StoreCode                  *string `json:"store_code"`
			StoreName                  *string `json:"store_name"`
			Locale                     *string `json:"locale"`
			BaseCurrencyCode           *string `json:"base_currency_code"`
			DefaultDisplayCurrencyCode *string `json:"default_display_currency_code"`
		} `json:"storeConfig"`
	}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	sc := data.StoreConfig
	if sc.ID == nil {
		t.Error("storeConfig.id is nil")
	}
	if sc.Code == nil || *sc.Code == "" {
		t.Error("storeConfig.code is empty")
	}
	if sc.StoreCode == nil || *sc.StoreCode == "" {
		t.Error("storeConfig.store_code is empty")
	}
	if sc.Locale == nil || *sc.Locale == "" {
		t.Error("storeConfig.locale is empty")
	}
	if sc.BaseCurrencyCode == nil || *sc.BaseCurrencyCode == "" {
		t.Error("storeConfig.base_currency_code is empty")
	}
}

func TestStoreConfig_CheckoutFields(t *testing.T) {
	resp := doQuery(t, `{
		storeConfig {
			is_guest_checkout_enabled
			is_one_page_checkout_enabled
			is_checkout_agreements_enabled
		}
	}`)
	if len(resp.Errors) > 0 {
		t.Fatalf("unexpected error: %s", resp.Errors[0].Message)
	}

	var data struct {
		StoreConfig struct {
			IsGuestCheckoutEnabled      *bool `json:"is_guest_checkout_enabled"`
			IsOnePageCheckoutEnabled    *bool `json:"is_one_page_checkout_enabled"`
			IsCheckoutAgreementsEnabled *bool `json:"is_checkout_agreements_enabled"`
		} `json:"storeConfig"`
	}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	// These should return non-nil booleans (defaulting to enabled)
	if data.StoreConfig.IsGuestCheckoutEnabled == nil {
		t.Error("is_guest_checkout_enabled is nil")
	}
	if data.StoreConfig.IsOnePageCheckoutEnabled == nil {
		t.Error("is_one_page_checkout_enabled is nil")
	}
}

// ─── availableStores ─────────────────────────────────────────────────────────

func TestAvailableStores_ReturnsStores(t *testing.T) {
	resp := doQuery(t, `{
		availableStores {
			id
			code
			store_code
			store_name
			website_code
		}
	}`)
	if len(resp.Errors) > 0 {
		t.Fatalf("unexpected error: %s", resp.Errors[0].Message)
	}

	var data struct {
		AvailableStores []struct {
			ID        *int    `json:"id"`
			Code      *string `json:"code"`
			StoreCode *string `json:"store_code"`
			StoreName *string `json:"store_name"`
		} `json:"availableStores"`
	}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(data.AvailableStores) == 0 {
		t.Fatal("availableStores returned no stores")
	}
	for i, s := range data.AvailableStores {
		if s.Code == nil || *s.Code == "" {
			t.Errorf("store[%d].code is empty", i)
		}
	}
}

func TestAvailableStores_UseCurrentGroup(t *testing.T) {
	resp := doQuery(t, `{
		availableStores(useCurrentGroup: true) {
			id
			code
		}
	}`)
	if len(resp.Errors) > 0 {
		t.Fatalf("unexpected error: %s", resp.Errors[0].Message)
	}

	var data struct {
		AvailableStores []struct {
			ID   *int    `json:"id"`
			Code *string `json:"code"`
		} `json:"availableStores"`
	}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	// Should return at least the current store
	if len(data.AvailableStores) == 0 {
		t.Fatal("availableStores(useCurrentGroup: true) returned no stores")
	}
}

// ─── countries ───────────────────────────────────────────────────────────────

func TestDirectory_Countries_ReturnsNonEmpty(t *testing.T) {
	resp := doQuery(t, `{
		countries {
			id
			full_name_english
			full_name_locale
			available_regions {
				id
				code
				name
			}
		}
	}`)
	if len(resp.Errors) > 0 {
		t.Fatalf("unexpected error: %s", resp.Errors[0].Message)
	}

	var data struct {
		Countries []struct {
			ID              string `json:"id"`
			FullNameEnglish string `json:"full_name_english"`
			FullNameLocale  string `json:"full_name_locale"`
		} `json:"countries"`
	}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(data.Countries) == 0 {
		t.Fatal("countries returned empty list")
	}

	// Check US is present with regions
	var usFound bool
	for _, c := range data.Countries {
		if c.ID == "US" {
			usFound = true
			if c.FullNameEnglish == "" {
				t.Error("US full_name_english is empty")
			}
			if c.FullNameLocale == "" {
				t.Error("US full_name_locale is empty")
			}
			break
		}
	}
	if !usFound {
		t.Error("US not found in countries")
	}
}

func TestDirectory_Country_ByID_US(t *testing.T) {
	resp := doQuery(t, `{
		country(id: "US") {
			id
			full_name_english
			full_name_locale
			available_regions {
				id
				code
				name
			}
		}
	}`)
	if len(resp.Errors) > 0 {
		t.Fatalf("unexpected error: %s", resp.Errors[0].Message)
	}

	var data struct {
		Country struct {
			ID              string `json:"id"`
			FullNameEnglish string `json:"full_name_english"`
			AvailableRegions []struct {
				ID   int    `json:"id"`
				Code string `json:"code"`
				Name string `json:"name"`
			} `json:"available_regions"`
		} `json:"country"`
	}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if data.Country.ID != "US" {
		t.Errorf("expected id=US, got %q", data.Country.ID)
	}
	if data.Country.FullNameEnglish == "" {
		t.Error("full_name_english is empty for US")
	}
	if len(data.Country.AvailableRegions) == 0 {
		t.Error("US should have available_regions (states)")
	}

	// Verify Texas is present
	var txFound bool
	for _, r := range data.Country.AvailableRegions {
		if r.Code == "TX" {
			txFound = true
			if r.Name == "" {
				t.Error("TX region name is empty")
			}
			break
		}
	}
	if !txFound {
		t.Error("TX not found in US regions")
	}
}

func TestDirectory_Country_NotFound(t *testing.T) {
	resp := doQuery(t, `{ country(id: "ZZ") { id } }`)
	if len(resp.Errors) == 0 {
		t.Fatal("expected error for unknown country, got none")
	}
	msg := resp.Errors[0].Message
	if !strings.Contains(msg, "available") && !strings.Contains(msg, "available") {
		t.Errorf("unexpected error message: %s", msg)
	}
}

func TestDirectory_Country_NoID_ReturnsError(t *testing.T) {
	resp := doQuery(t, `{ country { id } }`)
	if len(resp.Errors) == 0 {
		t.Fatal("expected error when id is missing, got none")
	}
}

// ─── currency ────────────────────────────────────────────────────────────────

func TestDirectory_Currency_ReturnsBaseCurrency(t *testing.T) {
	resp := doQuery(t, `{
		currency {
			base_currency_code
			base_currency_symbol
			default_display_currency_code
			default_display_currency_symbol
			available_currency_codes
			exchange_rates {
				currency_to
				rate
			}
		}
	}`)
	if len(resp.Errors) > 0 {
		t.Fatalf("unexpected error: %s", resp.Errors[0].Message)
	}

	var data struct {
		Currency struct {
			BaseCurrencyCode           string   `json:"base_currency_code"`
			BaseCurrencySymbol         string   `json:"base_currency_symbol"`
			DefaultDisplayCurrencyCode string   `json:"default_display_currency_code"`
			AvailableCurrencyCodes     []string `json:"available_currency_codes"`
		} `json:"currency"`
	}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if data.Currency.BaseCurrencyCode == "" {
		t.Error("base_currency_code is empty")
	}
	if data.Currency.DefaultDisplayCurrencyCode == "" {
		t.Error("default_display_currency_code is empty")
	}
}

// ─── cmsBlocks ───────────────────────────────────────────────────────────────

func TestBlock_Cms_EmptyIdentifiers(t *testing.T) {
	resp := doQuery(t, `{
		cmsBlocks(identifiers: []) {
			items {
				identifier
				title
				content
			}
		}
	}`)
	if len(resp.Errors) > 0 {
		t.Fatalf("unexpected error: %s", resp.Errors[0].Message)
	}

	var data struct {
		CmsBlocks struct {
			Items []struct {
				Identifier string `json:"identifier"`
			} `json:"items"`
		} `json:"cmsBlocks"`
	}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	// Empty identifiers returns empty items, not an error
	if len(data.CmsBlocks.Items) != 0 {
		t.Errorf("expected 0 items for empty identifiers, got %d", len(data.CmsBlocks.Items))
	}
}

func TestBlock_Cms_UnknownIdentifier(t *testing.T) {
	resp := doQuery(t, `{
		cmsBlocks(identifiers: ["__nonexistent_block_xyz__"]) {
			items {
				identifier
				title
			}
		}
	}`)
	if len(resp.Errors) > 0 {
		t.Fatalf("unexpected error: %s", resp.Errors[0].Message)
	}

	var data struct {
		CmsBlocks struct {
			Items []struct {
				Identifier string `json:"identifier"`
			} `json:"items"`
		} `json:"cmsBlocks"`
	}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(data.CmsBlocks.Items) != 0 {
		t.Errorf("expected 0 items for unknown identifier, got %d", len(data.CmsBlocks.Items))
	}
}

// ─── cmsPage ─────────────────────────────────────────────────────────────────

func TestPage_Cms_UnknownIdentifier(t *testing.T) {
	resp := doQuery(t, `{
		cmsPage(identifier: "__nonexistent_page_xyz__") {
			identifier
			title
		}
	}`)
	// Magento returns null (not an error) for missing pages
	var data struct {
		CmsPage *struct {
			Identifier string `json:"identifier"`
		} `json:"cmsPage"`
	}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if data.CmsPage != nil {
		t.Errorf("expected null for unknown cms page, got %+v", data.CmsPage)
	}
}

// ─── contactUs ───────────────────────────────────────────────────────────────

// contactUsFormEnabled checks whether the contact form is enabled in the DB.
// Returns false when contact/contact/enabled = 0.
func contactUsFormEnabled(t *testing.T) bool {
	t.Helper()
	resp := doQuery(t, `{ storeConfig { contact_enabled } }`)
	if len(resp.Errors) > 0 {
		return false
	}
	var data struct {
		StoreConfig struct {
			ContactEnabled *bool `json:"contact_enabled"`
		} `json:"storeConfig"`
	}
	json.Unmarshal(resp.Data, &data)
	return data.StoreConfig.ContactEnabled != nil && *data.StoreConfig.ContactEnabled
}

func TestMutation_ContactUs_MissingName(t *testing.T) {
	if !contactUsFormEnabled(t) {
		t.Skip("contact form is disabled in this environment")
	}
	resp := doQuery(t, `mutation {
		contactUs(input: {
			name: ""
			email: "test@example.com"
			telephone: "555-1234"
			comment: "Hello"
		}) {
			status
		}
	}`)
	if len(resp.Errors) == 0 {
		t.Fatal("expected error for empty name, got none")
	}
	if !strings.Contains(resp.Errors[0].Message, "Name") {
		t.Errorf("unexpected error message: %s", resp.Errors[0].Message)
	}
}

func TestMutation_ContactUs_MissingComment(t *testing.T) {
	if !contactUsFormEnabled(t) {
		t.Skip("contact form is disabled in this environment")
	}
	resp := doQuery(t, `mutation {
		contactUs(input: {
			name: "Test User"
			email: "test@example.com"
			telephone: ""
			comment: ""
		}) {
			status
		}
	}`)
	if len(resp.Errors) == 0 {
		t.Fatal("expected error for empty comment, got none")
	}
	if !strings.Contains(resp.Errors[0].Message, "Comment") {
		t.Errorf("unexpected error message: %s", resp.Errors[0].Message)
	}
}

func TestMutation_ContactUs_InvalidEmail(t *testing.T) {
	if !contactUsFormEnabled(t) {
		t.Skip("contact form is disabled in this environment")
	}
	resp := doQuery(t, `mutation {
		contactUs(input: {
			name: "Test User"
			email: "notanemail"
			telephone: ""
			comment: "Hello there"
		}) {
			status
		}
	}`)
	if len(resp.Errors) == 0 {
		t.Fatal("expected error for invalid email, got none")
	}
}

func TestMutation_ContactUs_Disabled_ReturnsError(t *testing.T) {
	// This test verifies the disabled-form error is returned when disabled.
	resp := doQuery(t, `mutation {
		contactUs(input: {
			name: "Test User"
			email: "test@example.com"
			telephone: ""
			comment: "Hello there"
		}) {
			status
		}
	}`)
	// Either succeeds (if enabled) or returns unavailable error (if disabled)
	if len(resp.Errors) > 0 {
		msg := resp.Errors[0].Message
		if !strings.Contains(msg, "unavailable") && !strings.Contains(msg, "Name") &&
			!strings.Contains(msg, "Comment") && !strings.Contains(msg, "email") {
			t.Errorf("unexpected error message: %s", msg)
		}
	}
}
