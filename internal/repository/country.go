package repository

import (
	"context"
	"database/sql"
	"fmt"
)

// RegionRow holds a single row from directory_country_region.
type RegionRow struct {
	RegionID  int
	CountryID string
	Code      string
	Name      string
}

// CountryRow holds a row from directory_country plus its regions.
type CountryRow struct {
	CountryID string // ISO2 code
	Iso2Code  string
	Iso3Code  string
	Regions   []*RegionRow
}

// CountryRepository loads country and region data.
type CountryRepository struct {
	db *sql.DB
}

// NewCountryRepository creates a CountryRepository.
func NewCountryRepository(db *sql.DB) *CountryRepository {
	return &CountryRepository{db: db}
}

// GetAll returns all countries with their regions.
func (r *CountryRepository) GetAll(ctx context.Context) ([]*CountryRow, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT country_id, COALESCE(iso2_code,''), COALESCE(iso3_code,'')
		 FROM directory_country ORDER BY country_id`,
	)
	if err != nil {
		return nil, fmt.Errorf("GetAll countries: %w", err)
	}
	defer rows.Close()

	var countries []*CountryRow
	index := make(map[string]*CountryRow)
	for rows.Next() {
		c := &CountryRow{}
		if err := rows.Scan(&c.CountryID, &c.Iso2Code, &c.Iso3Code); err != nil {
			return nil, err
		}
		countries = append(countries, c)
		index[c.CountryID] = c
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Load all regions and distribute
	regionRows, err := r.allRegions(ctx)
	if err != nil {
		return nil, err
	}
	for _, reg := range regionRows {
		if c, ok := index[reg.CountryID]; ok {
			c.Regions = append(c.Regions, reg)
		}
	}

	return countries, nil
}

// GetByID returns a single country with its regions.
func (r *CountryRepository) GetByID(ctx context.Context, id string) (*CountryRow, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT country_id, COALESCE(iso2_code,''), COALESCE(iso3_code,'')
		 FROM directory_country WHERE country_id = ?`, id,
	)
	c := &CountryRow{}
	if err := row.Scan(&c.CountryID, &c.Iso2Code, &c.Iso3Code); err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("GetByID country %s: %w", id, err)
	}

	regs, err := r.regionsForCountry(ctx, id)
	if err != nil {
		return nil, err
	}
	c.Regions = regs
	return c, nil
}

func (r *CountryRepository) allRegions(ctx context.Context) ([]*RegionRow, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT region_id, country_id, COALESCE(code,''), COALESCE(default_name,'')
		 FROM directory_country_region ORDER BY country_id, region_id`,
	)
	if err != nil {
		return nil, fmt.Errorf("allRegions: %w", err)
	}
	defer rows.Close()
	return scanRegionRows(rows)
}

func (r *CountryRepository) regionsForCountry(ctx context.Context, countryID string) ([]*RegionRow, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT region_id, country_id, COALESCE(code,''), COALESCE(default_name,'')
		 FROM directory_country_region WHERE country_id = ? ORDER BY region_id`, countryID,
	)
	if err != nil {
		return nil, fmt.Errorf("regionsForCountry %s: %w", countryID, err)
	}
	defer rows.Close()
	return scanRegionRows(rows)
}

func scanRegionRows(rows *sql.Rows) ([]*RegionRow, error) {
	var result []*RegionRow
	for rows.Next() {
		reg := &RegionRow{}
		if err := rows.Scan(&reg.RegionID, &reg.CountryID, &reg.Code, &reg.Name); err != nil {
			return nil, err
		}
		result = append(result, reg)
	}
	return result, rows.Err()
}
