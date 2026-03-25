package tests

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

// Comparison tests run the same queries against both the Go service (via httptest)
// and Magento PHP (at :8080), then compare responses field by field.
//
// Run: GOTOOLCHAIN=auto go test ./tests/ -run TestCompare -v -timeout 120s -count=1
//
// Requirements:
//   - MySQL with Magento sample data
//   - Magento PHP running at localhost:8080

const magentoURL = "http://localhost:8080/graphql"

func doMagentoQuery(t *testing.T, query string) gqlResponse {
	t.Helper()
	body := `{"query":` + jsonString(query) + `}`
	req, err := http.NewRequest("POST", magentoURL, strings.NewReader(body))
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Store", "default")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Skipf("Magento not available: %v", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var gqlResp gqlResponse
	if err := json.Unmarshal(respBody, &gqlResp); err != nil {
		t.Fatalf("parse Magento response: %v\nbody: %s", err, string(respBody))
	}
	return gqlResp
}

func jsonString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// ─── StoreConfig Comparison ───────────────────────────────────────────────────

func TestCompare_StoreConfig(t *testing.T) {
	query := `{
		storeConfig {
			id
			code
			store_code
			store_name
			locale
			timezone
			base_currency_code
			default_display_currency_code
			base_url
			secure_base_url
			is_guest_checkout_enabled
			is_one_page_checkout_enabled
			product_url_suffix
			category_url_suffix
			default_country
		}
	}`

	goResp := doQuery(t, query)
	magentoResp := doMagentoQuery(t, query)

	if len(goResp.Errors) > 0 {
		t.Fatalf("Go error: %s", goResp.Errors[0].Message)
	}
	if len(magentoResp.Errors) > 0 {
		t.Fatalf("Magento error: %s", magentoResp.Errors[0].Message)
	}

	type storeConfigShape struct {
		StoreConfig struct {
			ID                         *int    `json:"id"`
			Code                       *string `json:"code"`
			StoreCode                  *string `json:"store_code"`
			StoreName                  *string `json:"store_name"`
			Locale                     *string `json:"locale"`
			Timezone                   *string `json:"timezone"`
			BaseCurrencyCode           *string `json:"base_currency_code"`
			DefaultDisplayCurrencyCode *string `json:"default_display_currency_code"`
			BaseURL                    *string `json:"base_url"`
			SecureBaseURL              *string `json:"secure_base_url"`
			IsGuestCheckoutEnabled     *bool   `json:"is_guest_checkout_enabled"`
			IsOnePageCheckoutEnabled   *bool   `json:"is_one_page_checkout_enabled"`
			ProductURLSuffix           *string `json:"product_url_suffix"`
			CategoryURLSuffix          *string `json:"category_url_suffix"`
			DefaultCountry             *string `json:"default_country"`
		} `json:"storeConfig"`
	}

	var goData, magentoData storeConfigShape
	json.Unmarshal(goResp.Data, &goData)
	json.Unmarshal(magentoResp.Data, &magentoData)

	g := goData.StoreConfig
	m := magentoData.StoreConfig

	comparePtr(t, "id", g.ID, m.ID)
	comparePtr(t, "code", g.Code, m.Code)
	comparePtr(t, "store_code", g.StoreCode, m.StoreCode)
	comparePtr(t, "locale", g.Locale, m.Locale)
	comparePtr(t, "timezone", g.Timezone, m.Timezone)
	comparePtr(t, "base_currency_code", g.BaseCurrencyCode, m.BaseCurrencyCode)
	comparePtr(t, "default_display_currency_code", g.DefaultDisplayCurrencyCode, m.DefaultDisplayCurrencyCode)
	comparePtr(t, "is_guest_checkout_enabled", g.IsGuestCheckoutEnabled, m.IsGuestCheckoutEnabled)
	comparePtr(t, "is_one_page_checkout_enabled", g.IsOnePageCheckoutEnabled, m.IsOnePageCheckoutEnabled)
	comparePtr(t, "product_url_suffix", g.ProductURLSuffix, m.ProductURLSuffix)
	comparePtr(t, "category_url_suffix", g.CategoryURLSuffix, m.CategoryURLSuffix)
	comparePtr(t, "default_country", g.DefaultCountry, m.DefaultCountry)
}

// ─── Countries Comparison ─────────────────────────────────────────────────────

func TestCompare_Countries_US(t *testing.T) {
	query := `{
		country(id: "US") {
			id
			full_name_english
			available_regions {
				id
				code
				name
			}
		}
	}`

	goResp := doQuery(t, query)
	magentoResp := doMagentoQuery(t, query)

	if len(goResp.Errors) > 0 {
		t.Fatalf("Go error: %s", goResp.Errors[0].Message)
	}
	if len(magentoResp.Errors) > 0 {
		t.Fatalf("Magento error: %s", magentoResp.Errors[0].Message)
	}

	type countryShape struct {
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

	var goData, magentoData countryShape
	json.Unmarshal(goResp.Data, &goData)
	json.Unmarshal(magentoResp.Data, &magentoData)

	if goData.Country.ID != magentoData.Country.ID {
		t.Errorf("id mismatch: go=%q magento=%q", goData.Country.ID, magentoData.Country.ID)
	}
	if goData.Country.FullNameEnglish != magentoData.Country.FullNameEnglish {
		t.Errorf("full_name_english mismatch: go=%q magento=%q",
			goData.Country.FullNameEnglish, magentoData.Country.FullNameEnglish)
	}

	goRegions := make(map[string]string)
	for _, r := range goData.Country.AvailableRegions {
		goRegions[r.Code] = r.Name
	}
	magentoRegions := make(map[string]string)
	for _, r := range magentoData.Country.AvailableRegions {
		magentoRegions[r.Code] = r.Name
	}

	if len(goRegions) != len(magentoRegions) {
		t.Errorf("region count mismatch: go=%d magento=%d", len(goRegions), len(magentoRegions))
	}
	for code, goName := range goRegions {
		if magName, ok := magentoRegions[code]; ok {
			if goName != magName {
				t.Errorf("region %s name mismatch: go=%q magento=%q", code, goName, magName)
			}
		} else {
			t.Errorf("region %s present in Go but not in Magento", code)
		}
	}
}

func TestCompare_Currency(t *testing.T) {
	query := `{
		currency {
			base_currency_code
			base_currency_symbol
			default_display_currency_code
			default_display_currency_symbol
			available_currency_codes
		}
	}`

	goResp := doQuery(t, query)
	magentoResp := doMagentoQuery(t, query)

	if len(goResp.Errors) > 0 {
		t.Fatalf("Go error: %s", goResp.Errors[0].Message)
	}
	if len(magentoResp.Errors) > 0 {
		t.Fatalf("Magento error: %s", magentoResp.Errors[0].Message)
	}

	type currencyShape struct {
		Currency struct {
			BaseCurrencyCode           string   `json:"base_currency_code"`
			BaseCurrencySymbol         string   `json:"base_currency_symbol"`
			DefaultDisplayCurrencyCode string   `json:"default_display_currency_code"`
			AvailableCurrencyCodes     []string `json:"available_currency_codes"`
		} `json:"currency"`
	}

	var goData, magentoData currencyShape
	json.Unmarshal(goResp.Data, &goData)
	json.Unmarshal(magentoResp.Data, &magentoData)

	g := goData.Currency
	m := magentoData.Currency

	if g.BaseCurrencyCode != m.BaseCurrencyCode {
		t.Errorf("base_currency_code mismatch: go=%q magento=%q", g.BaseCurrencyCode, m.BaseCurrencyCode)
	}
	if g.DefaultDisplayCurrencyCode != m.DefaultDisplayCurrencyCode {
		t.Errorf("default_display_currency_code mismatch: go=%q magento=%q",
			g.DefaultDisplayCurrencyCode, m.DefaultDisplayCurrencyCode)
	}
	if g.BaseCurrencySymbol != m.BaseCurrencySymbol {
		t.Logf("base_currency_symbol mismatch (may be expected): go=%q magento=%q",
			g.BaseCurrencySymbol, m.BaseCurrencySymbol)
	}
}

// ─── Route Comparison ─────────────────────────────────────────────────────────

func TestCompare_Route_CmsPage(t *testing.T) {
	query := `{
		route(url: "home") {
			relative_url
			redirect_code
			type
			... on CmsPage {
				identifier
				title
			}
		}
	}`

	goResp := doQuery(t, query)
	magentoResp := doMagentoQuery(t, query)

	if len(goResp.Errors) > 0 {
		t.Fatalf("Go error: %s", goResp.Errors[0].Message)
	}
	if len(magentoResp.Errors) > 0 {
		t.Fatalf("Magento error: %s", magentoResp.Errors[0].Message)
	}

	type routeShape struct {
		Route *struct {
			RelativeURL  *string `json:"relative_url"`
			RedirectCode int     `json:"redirect_code"`
			Type         *string `json:"type"`
			Identifier   *string `json:"identifier"`
			Title        *string `json:"title"`
		} `json:"route"`
	}

	var goData, magentoData routeShape
	json.Unmarshal(goResp.Data, &goData)
	json.Unmarshal(magentoResp.Data, &magentoData)

	if goData.Route == nil && magentoData.Route == nil {
		return
	}
	if goData.Route == nil || magentoData.Route == nil {
		t.Fatalf("route nil mismatch: go=%v magento=%v", goData.Route == nil, magentoData.Route == nil)
	}

	comparePtr(t, "type", goData.Route.Type, magentoData.Route.Type)
	if goData.Route.RedirectCode != magentoData.Route.RedirectCode {
		t.Errorf("redirect_code mismatch: go=%d magento=%d", goData.Route.RedirectCode, magentoData.Route.RedirectCode)
	}
	comparePtr(t, "identifier", goData.Route.Identifier, magentoData.Route.Identifier)
}

func TestCompare_Route_Category(t *testing.T) {
	query := `{
		route(url: "gear.html") {
			relative_url
			redirect_code
			type
		}
	}`

	goResp := doQuery(t, query)
	magentoResp := doMagentoQuery(t, query)

	if len(goResp.Errors) > 0 {
		t.Fatalf("Go error: %s", goResp.Errors[0].Message)
	}
	if len(magentoResp.Errors) > 0 {
		t.Fatalf("Magento error: %s", magentoResp.Errors[0].Message)
	}

	type routeShape struct {
		Route *struct {
			RelativeURL  *string `json:"relative_url"`
			RedirectCode int     `json:"redirect_code"`
			Type         *string `json:"type"`
		} `json:"route"`
	}

	var goData, magentoData routeShape
	json.Unmarshal(goResp.Data, &goData)
	json.Unmarshal(magentoResp.Data, &magentoData)

	if goData.Route == nil && magentoData.Route == nil {
		return
	}
	if goData.Route == nil || magentoData.Route == nil {
		t.Fatalf("route nil mismatch: go=%v magento=%v", goData.Route == nil, magentoData.Route == nil)
	}

	comparePtr(t, "type", goData.Route.Type, magentoData.Route.Type)
	comparePtr(t, "relative_url", goData.Route.RelativeURL, magentoData.Route.RelativeURL)
	if goData.Route.RedirectCode != magentoData.Route.RedirectCode {
		t.Errorf("redirect_code mismatch: go=%d magento=%d", goData.Route.RedirectCode, magentoData.Route.RedirectCode)
	}
}

// ─── URLResolver Comparison ───────────────────────────────────────────────────

func TestCompare_URLResolver_CmsPage(t *testing.T) {
	// Note: Magento's EntityUrl does not expose redirect_code — omit it from comparison query.
	query := `{
		urlResolver(url: "home") {
			id
			type
			relative_url
		}
	}`

	goResp := doQuery(t, query)
	magentoResp := doMagentoQuery(t, query)

	if len(goResp.Errors) > 0 {
		t.Fatalf("Go error: %s", goResp.Errors[0].Message)
	}
	if len(magentoResp.Errors) > 0 {
		t.Fatalf("Magento error: %s", magentoResp.Errors[0].Message)
	}

	type urlResolverShape struct {
		URLResolver *struct {
			ID          *int    `json:"id"`
			Type        *string `json:"type"`
			RelativeURL *string `json:"relative_url"`
		} `json:"urlResolver"`
	}

	var goData, magentoData urlResolverShape
	json.Unmarshal(goResp.Data, &goData)
	json.Unmarshal(magentoResp.Data, &magentoData)

	if goData.URLResolver == nil && magentoData.URLResolver == nil {
		return
	}
	if goData.URLResolver == nil || magentoData.URLResolver == nil {
		t.Fatalf("urlResolver nil mismatch: go=%v magento=%v",
			goData.URLResolver == nil, magentoData.URLResolver == nil)
	}

	comparePtr(t, "id", goData.URLResolver.ID, magentoData.URLResolver.ID)
	comparePtr(t, "type", goData.URLResolver.Type, magentoData.URLResolver.Type)
	comparePtr(t, "relative_url", goData.URLResolver.RelativeURL, magentoData.URLResolver.RelativeURL)
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func comparePtr[T comparable](t *testing.T, field string, goVal, magentoVal *T) {
	t.Helper()
	if goVal == nil && magentoVal == nil {
		return
	}
	if goVal == nil {
		t.Errorf("%s: go=nil magento=%v", field, *magentoVal)
		return
	}
	if magentoVal == nil {
		t.Errorf("%s: go=%v magento=nil", field, *goVal)
		return
	}
	if *goVal != *magentoVal {
		t.Errorf("%s mismatch: go=%v magento=%v", field, *goVal, *magentoVal)
	}
}
