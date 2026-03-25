package repository

import (
	"context"
	"database/sql"
	"fmt"
)

// StoreRow holds joined data from store, store_group, and store_website.
type StoreRow struct {
	StoreID               int
	StoreCode             string
	StoreName             string
	StoreSortOrder        int
	GroupID               int
	GroupCode             string
	GroupName             string
	GroupDefaultStoreID   int
	WebsiteID             int
	WebsiteCode           string
	WebsiteName           string
	WebsiteDefaultGroupID int
	WebsiteIsDefault      bool
}

// StoreRepository loads store data with joined group and website rows.
type StoreRepository struct {
	db *sql.DB
}

// NewStoreRepository creates a StoreRepository.
func NewStoreRepository(db *sql.DB) *StoreRepository {
	return &StoreRepository{db: db}
}

const storeSelectCols = `
	s.store_id, s.code, s.name, COALESCE(s.sort_order,0),
	sg.group_id, COALESCE(sg.code,''), COALESCE(sg.name,''), COALESCE(sg.default_store_id,0),
	sw.website_id, COALESCE(sw.code,''), COALESCE(sw.name,''), COALESCE(sw.default_group_id,0),
	COALESCE(sw.is_default,0)
`

const storeJoins = `
	FROM store s
	JOIN store_group sg ON sg.group_id = s.group_id
	JOIN store_website sw ON sw.website_id = s.website_id
`

func scanStoreRow(row interface{ Scan(...any) error }) (*StoreRow, error) {
	var r StoreRow
	var isDefault int
	err := row.Scan(
		&r.StoreID, &r.StoreCode, &r.StoreName, &r.StoreSortOrder,
		&r.GroupID, &r.GroupCode, &r.GroupName, &r.GroupDefaultStoreID,
		&r.WebsiteID, &r.WebsiteCode, &r.WebsiteName, &r.WebsiteDefaultGroupID,
		&isDefault,
	)
	if err != nil {
		return nil, err
	}
	r.WebsiteIsDefault = isDefault == 1
	return &r, nil
}

// GetByID loads a single active store row by store_id.
func (r *StoreRepository) GetByID(ctx context.Context, storeID int) (*StoreRow, error) {
	q := `SELECT` + storeSelectCols + storeJoins + `WHERE s.store_id = ? AND s.is_active = 1`
	row := r.db.QueryRowContext(ctx, q, storeID)
	sr, err := scanStoreRow(row)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("store %d not found", storeID)
	}
	return sr, err
}

// GetByWebsite loads all active stores for a website.
func (r *StoreRepository) GetByWebsite(ctx context.Context, websiteID int) ([]*StoreRow, error) {
	q := `SELECT` + storeSelectCols + storeJoins + `WHERE sw.website_id = ? AND s.is_active = 1 ORDER BY s.sort_order, s.store_id`
	rows, err := r.db.QueryContext(ctx, q, websiteID)
	if err != nil {
		return nil, fmt.Errorf("GetByWebsite: %w", err)
	}
	defer rows.Close()
	return scanStoreRows(rows)
}

// GetByGroup loads all active stores for a store group.
func (r *StoreRepository) GetByGroup(ctx context.Context, groupID int) ([]*StoreRow, error) {
	q := `SELECT` + storeSelectCols + storeJoins + `WHERE sg.group_id = ? AND s.is_active = 1 ORDER BY s.sort_order, s.store_id`
	rows, err := r.db.QueryContext(ctx, q, groupID)
	if err != nil {
		return nil, fmt.Errorf("GetByGroup: %w", err)
	}
	defer rows.Close()
	return scanStoreRows(rows)
}

func scanStoreRows(rows *sql.Rows) ([]*StoreRow, error) {
	var result []*StoreRow
	for rows.Next() {
		sr, err := scanStoreRow(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, sr)
	}
	return result, rows.Err()
}

// GetDefaultStoreCodeForGroup returns the code of the default store in a group.
// Returns empty string if not found.
func (r *StoreRepository) GetDefaultStoreCodeForGroup(ctx context.Context, defaultStoreID int) string {
	var code string
	err := r.db.QueryRowContext(ctx,
		`SELECT code FROM store WHERE store_id = ? AND is_active = 1`, defaultStoreID,
	).Scan(&code)
	if err != nil {
		return ""
	}
	return code
}
