package service

import (
	"context"
	"fmt"
	"strings"

	"golang.org/x/text/language"
	"golang.org/x/text/language/display"

	"github.com/magendooro/magento2-go-common/config"
	"github.com/magendooro/magento2-store-graphql-go/graph/model"
	"github.com/magendooro/magento2-store-graphql-go/internal/repository"
)

// StoreService orchestrates data loading for all store/directory/CMS resolvers.
type StoreService struct {
	storeRepo    *repository.StoreRepository
	countryRepo  *repository.CountryRepository
	currencyRepo *repository.CurrencyRepository
	cmsRepo      *repository.CmsRepository
	cp           *config.ConfigProvider
}

// NewStoreService creates a StoreService.
func NewStoreService(
	storeRepo *repository.StoreRepository,
	countryRepo *repository.CountryRepository,
	currencyRepo *repository.CurrencyRepository,
	cmsRepo *repository.CmsRepository,
	cp *config.ConfigProvider,
) *StoreService {
	return &StoreService{
		storeRepo:    storeRepo,
		countryRepo:  countryRepo,
		currencyRepo: currencyRepo,
		cmsRepo:      cmsRepo,
		cp:           cp,
	}
}

// GetStoreConfig builds a full StoreConfig model for the given storeID.
func (s *StoreService) GetStoreConfig(ctx context.Context, storeID int) (*model.StoreConfig, error) {
	row, err := s.storeRepo.GetByID(ctx, storeID)
	if err != nil {
		return nil, fmt.Errorf("GetStoreConfig: %w", err)
	}

	// Determine whether this store is the default for its group
	defCode := s.storeRepo.GetDefaultStoreCodeForGroup(ctx, row.GroupDefaultStoreID)
	isDefaultStore := row.StoreCode == defCode

	// Determine whether this store's group is the default for the website
	// The website's default_group_id is stored in row.WebsiteDefaultGroupID
	isDefaultStoreGroup := row.GroupID == row.WebsiteDefaultGroupID

	cp := s.cp
	get := func(path string) *string { return strPtr(cp.Get(path, storeID)) }
	getDefault := func(path, def string) *string {
		if v := cp.Get(path, storeID); v != "" {
			return &v
		}
		return &def
	}
	// getBoolPtr reads a boolean config with an explicit default (1=true, 0=false).
	// Use this instead of cp.GetBool when the Magento default is "enabled" (1).
	getBoolPtr := func(path string, def int) *bool { v := cp.GetInt(path, storeID, def) == 1; return &v }
	getIntPtr := func(path string, def int) *int { v := cp.GetInt(path, storeID, def); return &v }

	logoWidth := cp.GetInt("design/header/logo_width", storeID, 0)
	logoHeight := cp.GetInt("design/header/logo_height", storeID, 0)
	demonotice := cp.GetInt("design/head/demonotice", storeID, 0)
	showCmsBreadcrumbs := cp.GetInt("web/default/show_cms_breadcrumbs", storeID, 1)
	maxItemsInOrderSummary := cp.GetInt("checkout/options/max_items_display_count", storeID, 10)
	minicartMaxItems := cp.GetInt("checkout/sidebar/count", storeID, 5)
	cartExpiresInDays := cp.GetInt("checkout/cart/delete_quote_after", storeID, 30)

	isCheckoutAgreementsEnabled := cp.GetBool("checkout/options/enable_agreements", storeID)
	contactEnabled := cp.GetBool("contact/contact/enabled", storeID)

	sortOrder := row.StoreSortOrder
	websiteID := row.WebsiteID

	return &model.StoreConfig{
		ID:                          &row.StoreID,
		Code:                        strPtr(row.StoreCode),
		StoreCode:                   strPtr(row.StoreCode),
		StoreName:                   strPtr(row.StoreName),
		StoreSortOrder:              &sortOrder,
		IsDefaultStore:              &isDefaultStore,
		StoreGroupCode:              strPtr(row.GroupCode),
		StoreGroupName:              strPtr(row.GroupName),
		IsDefaultStoreGroup:         &isDefaultStoreGroup,
		StoreGroupDefaultStoreCode:  strPtr(defCode),
		WebsiteID:                   &websiteID,
		WebsiteCode:                 strPtr(row.WebsiteCode),
		WebsiteName:                 strPtr(row.WebsiteName),
		Locale:                      get("general/locale/code"),
		Timezone:                    get("general/locale/timezone"),
		WeightUnit:                  get("general/locale/weight_unit"),
		BaseCurrencyCode:            get("currency/options/base"),
		DefaultDisplayCurrencyCode:  get("currency/options/default"),
		BaseURL:                     get("web/unsecure/base_url"),
		BaseLinkURL:                 get("web/unsecure/base_link_url"),
		BaseStaticURL:               get("web/unsecure/base_static_url"),
		BaseMediaURL:                get("web/unsecure/base_media_url"),
		SecureBaseURL:               get("web/secure/base_url"),
		SecureBaseLinkURL:           get("web/secure/base_link_url"),
		SecureBaseStaticURL:         get("web/secure/base_static_url"),
		SecureBaseMediaURL:          get("web/secure/base_media_url"),
		UseStoreInURL:               getBoolPtr("web/url/use_store", 0),
		DefaultCountry:              getDefault("general/country/default", "US"),
		CountriesWithRequiredRegion: get("general/region/state_required"),
		OptionalZipCountries:        get("general/country/optional_zip_countries"),
		DisplayStateIfOptional:      getBoolPtr("general/region/display_all", 0),
		DefaultTitle:                get("design/head/default_title"),
		TitlePrefix:                 get("design/head/title_prefix"),
		TitleSuffix:                 get("design/head/title_suffix"),
		DefaultDescription:          get("design/head/default_description"),
		DefaultKeywords:             get("design/head/default_keywords"),
		HeadShortcutIcon:            get("design/head/shortcut_icon"),
		HeaderLogoSrc:               get("design/header/logo_src"),
		LogoWidth:                   &logoWidth,
		LogoHeight:                  &logoHeight,
		LogoAlt:                     get("design/header/logo_alt"),
		Welcome:                     get("design/header/welcome"),
		Copyright:                   get("design/footer/copyright"),
		Demonotice:                  &demonotice,
		ProductURLSuffix:            getDefault("catalog/seo/product_url_suffix", ".html"),
		CategoryURLSuffix:           getDefault("catalog/seo/category_url_suffix", ".html"),
		TitleSeparator:              get("catalog/seo/title_separator"),
		ListMode:                    get("catalog/frontend/list_mode"),
		GridPerPageValues:           get("catalog/frontend/grid_per_page_values"),
		ListPerPageValues:           get("catalog/frontend/list_per_page_values"),
		GridPerPage:                 getIntPtr("catalog/frontend/grid_per_page", 9),
		ListPerPage:                 getIntPtr("catalog/frontend/list_per_page", 10),
		CatalogDefaultSortBy:        get("catalog/frontend/default_sort_by"),
		Front:                       get("web/default/front"),
		CmsHomePage:                 get("web/default/cms_home_page"),
		NoRoute:                     get("web/default/no_route"),
		CmsNoRoute:                  get("web/default/cms_no_route"),
		CmsNoCookies:                get("web/default/cms_no_cookies"),
		ShowCmsBreadcrumbs:          &showCmsBreadcrumbs,
		IsGuestCheckoutEnabled:      getBoolPtr("checkout/options/guest_checkout", 1),
		IsOnePageCheckoutEnabled:    getBoolPtr("checkout/options/onepage_checkout_enabled", 1),
		MaxItemsInOrderSummary:      &maxItemsInOrderSummary,
		MinicartDisplay:             getBoolPtr("checkout/sidebar/display", 1),
		MinicartMaxItems:            &minicartMaxItems,
		CartExpiresInDays:           &cartExpiresInDays,
		IsCheckoutAgreementsEnabled: isCheckoutAgreementsEnabled,
		MinimumPasswordLength:       get("customer/password/minimum_password_length"),
		RequiredCharacterClassesNumber: get("customer/password/required_character_classes_number"),
		AutocompleteOnStorefront:    getBoolPtr("customer/password/autocomplete_on_storefront", 0),
		CreateAccountConfirmation:   getBoolPtr("customer/create_account/confirm", 0),
		ContactEnabled:              contactEnabled,
	}, nil
}

// GetAvailableStores returns StoreConfig list for a website or group.
// Pass websiteID and groupID from the current store context.
func (s *StoreService) GetAvailableStores(ctx context.Context, websiteID int, groupID int, useCurrentGroup bool) ([]*model.StoreConfig, error) {
	var rows []*repository.StoreRow
	var err error
	if useCurrentGroup {
		rows, err = s.storeRepo.GetByGroup(ctx, groupID)
	} else {
		rows, err = s.storeRepo.GetByWebsite(ctx, websiteID)
	}
	if err != nil {
		return nil, fmt.Errorf("GetAvailableStores: %w", err)
	}

	result := make([]*model.StoreConfig, 0, len(rows))
	for _, row := range rows {
		sc, err := s.GetStoreConfig(ctx, row.StoreID)
		if err != nil {
			return nil, err
		}
		result = append(result, sc)
	}
	return result, nil
}

// GetCountries returns all countries with names in the store locale.
func (s *StoreService) GetCountries(ctx context.Context, locale string) ([]*model.Country, error) {
	rows, err := s.countryRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("GetCountries: %w", err)
	}

	localeTag := parseLocaleTag(locale)
	result := make([]*model.Country, 0, len(rows))
	for _, r := range rows {
		result = append(result, countryRowToModel(r, localeTag))
	}
	return result, nil
}

// GetCountry returns a single country by ISO2 code.
func (s *StoreService) GetCountry(ctx context.Context, id string, locale string) (*model.Country, error) {
	row, err := s.countryRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("GetCountry: %w", err)
	}
	if row == nil {
		return nil, nil
	}
	localeTag := parseLocaleTag(locale)
	return countryRowToModel(row, localeTag), nil
}

// GetCurrency returns the currency info for a store.
func (s *StoreService) GetCurrency(ctx context.Context, storeID int) (*model.Currency, error) {
	baseCurrency := s.cp.Get("currency/options/base", storeID)
	displayCurrency := s.cp.Get("currency/options/default", storeID)

	rates, err := s.currencyRepo.GetExchangeRates(ctx, baseCurrency)
	if err != nil {
		return nil, fmt.Errorf("GetCurrency: %w", err)
	}

	var availableCodes []*string
	var exchangeRates []*model.ExchangeRate
	for _, er := range rates {
		c := er.CurrencyTo
		availableCodes = append(availableCodes, strPtr(c))
		rate := er.Rate
		exchangeRates = append(exchangeRates, &model.ExchangeRate{
			CurrencyTo: strPtr(c),
			Rate:       &rate,
		})
	}

	baseSymbol := repository.SymbolFor(baseCurrency)
	displaySymbol := repository.SymbolFor(displayCurrency)

	return &model.Currency{
		BaseCurrencyCode:             strPtr(baseCurrency),
		BaseCurrencySymbol:           strPtr(baseSymbol),
		DefaultDisplayCurrencyCode:   strPtr(displayCurrency),
		DefaultDisplayCurrencySymbol: strPtr(displaySymbol),
		AvailableCurrencyCodes:       availableCodes,
		ExchangeRates:                exchangeRates,
	}, nil
}

// GetCmsBlocks returns CMS blocks by identifier, scoped to a store.
func (s *StoreService) GetCmsBlocks(ctx context.Context, storeID int, identifiers []string) (*model.CmsBlocks, error) {
	blocks, err := s.cmsRepo.GetBlocksByIdentifiers(ctx, storeID, identifiers)
	if err != nil {
		return nil, fmt.Errorf("GetCmsBlocks: %w", err)
	}

	items := make([]*model.CmsBlock, 0, len(blocks))
	for _, b := range blocks {
		items = append(items, &model.CmsBlock{
			Identifier: strPtr(b.Identifier),
			Title:      strPtr(b.Title),
			Content:    strPtr(b.Content),
		})
	}
	return &model.CmsBlocks{Items: items}, nil
}

// GetCmsPage returns a CMS page by id or identifier, scoped to a store.
func (s *StoreService) GetCmsPage(ctx context.Context, storeID int, id *int, identifier *string) (*model.CmsPage, error) {
	var page *repository.CmsPageData
	var err error

	switch {
	case identifier != nil && *identifier != "":
		page, err = s.cmsRepo.GetPageByIdentifier(ctx, storeID, *identifier)
	case id != nil:
		page, err = s.cmsRepo.GetPageByID(ctx, storeID, *id)
	default:
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("GetCmsPage: %w", err)
	}
	if page == nil {
		return nil, nil
	}

	return &model.CmsPage{
		Identifier:      strPtr(page.Identifier),
		URLKey:          strPtr(page.Identifier),
		Title:           strPtr(page.Title),
		Content:         strPtr(page.Content),
		ContentHeading:  strPtr(page.ContentHeading),
		PageLayout:      strPtr(page.PageLayout),
		MetaTitle:       strPtr(page.MetaTitle),
		MetaDescription: strPtr(page.MetaDescription),
		MetaKeywords:    strPtr(page.MetaKeywords),
	}, nil
}

// IsContactEnabled returns whether the contact form is enabled for a store.
func (s *StoreService) IsContactEnabled(storeID int) bool {
	return s.cp.GetBool("contact/contact/enabled", storeID)
}

// GetLocale returns the locale string for a store (e.g. "en_US").
func (s *StoreService) GetLocale(storeID int) string {
	return s.cp.Get("general/locale/code", storeID)
}

// GetStoreRow returns the raw store row (with group/website info) for a store ID.
func (s *StoreService) GetStoreRow(ctx context.Context, storeID int) (*repository.StoreRow, error) {
	return s.storeRepo.GetByID(ctx, storeID)
}

// ─── helpers ──────────────────────────────────────────────────────────────────

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func parseLocaleTag(locale string) language.Tag {
	if locale == "" {
		return language.English
	}
	return language.Make(strings.ReplaceAll(locale, "_", "-"))
}

func countryRowToModel(r *repository.CountryRow, localeTag language.Tag) *model.Country {
	iso2 := r.Iso2Code
	if iso2 == "" {
		iso2 = r.CountryID
	}

	nameEnglish := regionDisplayName(iso2, language.English)
	nameLocale := regionDisplayName(iso2, localeTag)

	regions := make([]*model.Region, 0, len(r.Regions))
	for _, reg := range r.Regions {
		regionID := reg.RegionID
		regions = append(regions, &model.Region{
			ID:   &regionID,
			Code: strPtr(reg.Code),
			Name: strPtr(reg.Name),
		})
	}

	var availableRegions []*model.Region
	if len(regions) > 0 {
		availableRegions = regions
	}

	return &model.Country{
		ID:                   strPtr(iso2),
		TwoLetterAbbreviation: strPtr(iso2),
		ThreeLetterAbbreviation: strPtr(r.Iso3Code),
		FullNameLocale:        strPtr(nameLocale),
		FullNameEnglish:       strPtr(nameEnglish),
		AvailableRegions:      availableRegions,
	}
}

func regionDisplayName(iso2 string, tag language.Tag) string {
	region, err := language.ParseRegion(iso2)
	if err != nil {
		return iso2
	}
	namer := display.Regions(tag)
	if namer == nil {
		return iso2
	}
	name := namer.Name(region)
	if name == "" {
		return iso2
	}
	return name
}
