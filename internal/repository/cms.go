package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// CmsBlockData holds a row from cms_block.
type CmsBlockData struct {
	BlockID    int
	Identifier string
	Title      string
	Content    string
	IsActive   bool
}

// CmsPageData holds a row from cms_page.
type CmsPageData struct {
	PageID          int
	Identifier      string
	Title           string
	Content         string
	ContentHeading  string
	PageLayout      string
	MetaTitle       string
	MetaDescription string
	MetaKeywords    string
	IsActive        bool
}

// CmsRepository loads CMS block and page data.
type CmsRepository struct {
	db *sql.DB
}

// NewCmsRepository creates a CmsRepository.
func NewCmsRepository(db *sql.DB) *CmsRepository {
	return &CmsRepository{db: db}
}

// GetBlocksByIdentifiers loads active CMS blocks for a store, preferring store-specific
// over store_id=0 (all-stores fallback).
func (r *CmsRepository) GetBlocksByIdentifiers(ctx context.Context, storeID int, identifiers []string) ([]*CmsBlockData, error) {
	if len(identifiers) == 0 {
		return nil, nil
	}

	placeholders := make([]string, len(identifiers))
	args := make([]interface{}, len(identifiers)+1)
	for i, id := range identifiers {
		placeholders[i] = "?"
		args[i] = id
	}
	args[len(identifiers)] = storeID

	q := fmt.Sprintf(`
		SELECT b.block_id, b.identifier, b.title, b.content
		FROM cms_block b
		INNER JOIN cms_block_store bs ON b.block_id = bs.block_id
		WHERE b.identifier IN (%s) AND b.is_active = 1 AND bs.store_id IN (0, ?)
		GROUP BY b.block_id
		ORDER BY MAX(bs.store_id) DESC
	`, strings.Join(placeholders, ","))

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("GetBlocksByIdentifiers: %w", err)
	}
	defer rows.Close()

	var result []*CmsBlockData
	for rows.Next() {
		b := &CmsBlockData{IsActive: true}
		if err := rows.Scan(&b.BlockID, &b.Identifier, &b.Title, &b.Content); err != nil {
			return nil, err
		}
		result = append(result, b)
	}
	return result, rows.Err()
}

// GetPageByIdentifier loads a CMS page by string identifier, scoped to the store.
func (r *CmsRepository) GetPageByIdentifier(ctx context.Context, storeID int, identifier string) (*CmsPageData, error) {
	return r.queryPage(ctx, storeID, "p.identifier = ?", identifier)
}

// GetPageByID loads a CMS page by integer ID, scoped to the store.
func (r *CmsRepository) GetPageByID(ctx context.Context, storeID int, pageID int) (*CmsPageData, error) {
	return r.queryPage(ctx, storeID, "p.page_id = ?", pageID)
}

func (r *CmsRepository) queryPage(ctx context.Context, storeID int, whereCond string, whereArg interface{}) (*CmsPageData, error) {
	q := fmt.Sprintf(`
		SELECT p.page_id, p.identifier, COALESCE(p.title,''), COALESCE(p.content,''),
		       COALESCE(p.content_heading,''), COALESCE(p.page_layout,''),
		       COALESCE(p.meta_title,''), COALESCE(p.meta_description,''), COALESCE(p.meta_keywords,''),
		       p.is_active
		FROM cms_page p
		INNER JOIN cms_page_store ps ON p.page_id = ps.page_id
		WHERE %s AND p.is_active = 1 AND ps.store_id IN (0, ?)
		ORDER BY ps.store_id DESC
		LIMIT 1
	`, whereCond)

	row := r.db.QueryRowContext(ctx, q, whereArg, storeID)
	p := &CmsPageData{}
	var isActive int
	err := row.Scan(
		&p.PageID, &p.Identifier, &p.Title, &p.Content,
		&p.ContentHeading, &p.PageLayout,
		&p.MetaTitle, &p.MetaDescription, &p.MetaKeywords,
		&isActive,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("queryPage: %w", err)
	}
	p.IsActive = isActive == 1
	return p, nil
}
