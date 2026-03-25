package repository

import (
	"context"
	"database/sql"
	"fmt"
)

// ExchangeRateRow holds a row from directory_currency_rate.
type ExchangeRateRow struct {
	CurrencyTo string
	Rate       float64
}

// CurrencyRepository loads currency exchange rate data.
type CurrencyRepository struct {
	db *sql.DB
}

// NewCurrencyRepository creates a CurrencyRepository.
func NewCurrencyRepository(db *sql.DB) *CurrencyRepository {
	return &CurrencyRepository{db: db}
}

// GetExchangeRates returns all exchange rates from the given base currency.
func (r *CurrencyRepository) GetExchangeRates(ctx context.Context, baseCurrency string) ([]*ExchangeRateRow, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT currency_to, COALESCE(rate, 0) FROM directory_currency_rate WHERE currency_from = ?`,
		baseCurrency,
	)
	if err != nil {
		return nil, fmt.Errorf("GetExchangeRates: %w", err)
	}
	defer rows.Close()

	var result []*ExchangeRateRow
	for rows.Next() {
		er := &ExchangeRateRow{}
		if err := rows.Scan(&er.CurrencyTo, &er.Rate); err != nil {
			return nil, err
		}
		result = append(result, er)
	}
	return result, rows.Err()
}

// CurrencySymbols maps ISO 4217 currency codes to their symbols.
var CurrencySymbols = map[string]string{
	"USD": "$",
	"EUR": "€",
	"GBP": "£",
	"JPY": "¥",
	"CAD": "CA$",
	"AUD": "A$",
	"CHF": "CHF",
	"CNY": "¥",
	"SEK": "kr",
	"NOK": "kr",
	"DKK": "kr",
	"NZD": "NZ$",
	"MXN": "MX$",
	"SGD": "S$",
	"HKD": "HK$",
	"INR": "₹",
	"BRL": "R$",
	"ZAR": "R",
	"RUB": "₽",
	"KRW": "₩",
	"TRY": "₺",
	"PLN": "zł",
	"CZK": "Kč",
	"HUF": "Ft",
	"RON": "lei",
	"BGN": "лв",
	"HRK": "kn",
	"ISK": "kr",
	"SAR": "﷼",
	"AED": "د.إ",
	"ILS": "₪",
	"PHP": "₱",
	"THB": "฿",
	"MYR": "RM",
	"IDR": "Rp",
	"VND": "₫",
	"UAH": "₴",
	"CLP": "CLP$",
	"ARS": "ARS$",
	"COP": "COL$",
	"PEN": "S/.",
	"EGP": "£",
	"NGN": "₦",
	"PKR": "₨",
	"BDT": "৳",
}

// SymbolFor returns the currency symbol for a code, falling back to the code itself.
func SymbolFor(code string) string {
	if s, ok := CurrencySymbols[code]; ok {
		return s
	}
	return code
}
