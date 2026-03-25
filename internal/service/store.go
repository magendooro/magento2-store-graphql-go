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
	urlRepo      *repository.UrlRepository
	cp           *config.ConfigProvider
}

// NewStoreService creates a StoreService.
func NewStoreService(
	storeRepo *repository.StoreRepository,
	countryRepo *repository.CountryRepository,
	currencyRepo *repository.CurrencyRepository,
	cmsRepo *repository.CmsRepository,
	urlRepo *repository.UrlRepository,
	cp *config.ConfigProvider,
) *StoreService {
	return &StoreService{
		storeRepo:    storeRepo,
		countryRepo:  countryRepo,
		currencyRepo: currencyRepo,
		cmsRepo:      cmsRepo,
		urlRepo:      urlRepo,
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

// ─── URL Routing ──────────────────────────────────────────────────────────────

// resolveUrlRewrite implements Magento's AbstractEntityUrl resolution logic:
// 1. Look up by request_path
// 2. If not found, try target_path reverse lookup
// 3. Follow redirect chain to the final destination
// Returns the final UrlRow, the redirect_code seen along the way, and the
// relative_url the client should use.
func (s *StoreService) resolveUrlRewrite(ctx context.Context, rawURL string, storeID int) (
	final *repository.UrlRow, redirectCode int, relativeURL string, err error,
) {
	path := repository.ParseURLPath(rawURL)

	row, err := s.urlRepo.GetByRequestPath(ctx, path, storeID)
	if err != nil {
		return nil, 0, "", err
	}
	if row == nil {
		// Reverse lookup: maybe the client sent a canonical target path
		row, err = s.urlRepo.GetByTargetPath(ctx, path, storeID)
		if err != nil {
			return nil, 0, "", err
		}
	}
	if row == nil {
		return nil, 0, "", nil
	}

	redirectCode = row.RedirectType
	relativeURL = row.RequestPath

	// Follow redirect chain (max 10 hops to prevent infinite loops)
	current := row
	for i := 0; i < 10 && current.RedirectType != 0; i++ {
		next, err := s.urlRepo.GetByRequestPath(ctx, current.TargetPath, storeID)
		if err != nil || next == nil {
			break
		}
		current = next
	}
	final = current

	// If the final row has no entity_id, try a target_path reverse lookup
	if final.EntityID == 0 {
		if candidate, err := s.urlRepo.GetByTargetPath(ctx, final.TargetPath, storeID); err == nil && candidate != nil {
			final = candidate
		}
	}

	if redirectCode > 0 {
		relativeURL = current.TargetPath
	}
	return final, redirectCode, relativeURL, nil
}

// GetRoute resolves a URL to a RoutableInterface implementation.
// For CMS pages the full page content is returned.
// For products and categories only routing metadata is returned;
// full entity data should be fetched from the catalog service.
func (s *StoreService) GetRoute(ctx context.Context, rawURL string, storeID int) (model.RoutableInterface, error) {
	if strings.TrimSpace(rawURL) == "" {
		return nil, fmt.Errorf("\"url\" argument should be specified and not empty")
	}

	final, redirectCode, relativeURL, err := s.resolveUrlRewrite(ctx, rawURL, storeID)
	if err != nil {
		return nil, err
	}
	if final == nil {
		return nil, nil
	}

	typeEnum := urlTypeEnum(final.EntityType)

	switch final.EntityType {
	case "cms-page":
		page, err := s.cmsRepo.GetPageByID(ctx, storeID, final.EntityID)
		if err != nil {
			return nil, err
		}
		if page == nil {
			return &model.CmsPage{
				RelativeURL:  &relativeURL,
				RedirectCode: redirectCode,
				Type:         typeEnum,
			}, nil
		}
		return cmsPageDataToModel(page, relativeURL, redirectCode, typeEnum), nil

	case "product":
		typeID, err := s.urlRepo.GetProductTypeID(ctx, final.EntityID)
		if err != nil {
			return nil, err
		}
		return productRoutable(typeID, relativeURL, redirectCode, typeEnum), nil

	case "category":
		uid := fmt.Sprintf("%d", final.EntityID)
		return &model.CategoryTree{
			RelativeURL:  &relativeURL,
			RedirectCode: redirectCode,
			Type:         typeEnum,
			UID:          &uid,
		}, nil

	default:
		return &model.RoutableURL{
			RelativeURL:  &relativeURL,
			RedirectCode: redirectCode,
			Type:         typeEnum,
		}, nil
	}
}

// GetUrlResolver implements the deprecated urlResolver query.
func (s *StoreService) GetUrlResolver(ctx context.Context, rawURL string, storeID int) (*model.EntityURL, error) {
	if strings.TrimSpace(rawURL) == "" {
		return nil, fmt.Errorf("\"url\" argument should be specified and not empty")
	}

	final, redirectCode, relativeURL, err := s.resolveUrlRewrite(ctx, rawURL, storeID)
	if err != nil {
		return nil, err
	}
	if final == nil {
		return nil, nil
	}

	typeEnum := urlTypeEnum(final.EntityType)
	uid := fmt.Sprintf("%d", final.EntityID)
	entityID := final.EntityID
	return &model.EntityURL{
		ID:           &entityID,
		EntityUID:    &uid,
		CanonicalURL: &relativeURL,
		RelativeURL:  &relativeURL,
		RedirectCode: &redirectCode,
		Type:         typeEnum,
	}, nil
}

// cmsPageDataToModel converts a repository CmsPageData row to a model.CmsPage
// with the RoutableInterface routing fields populated.
func cmsPageDataToModel(p *repository.CmsPageData, relativeURL string, redirectCode int, t *model.URLRewriteEntityTypeEnum) *model.CmsPage {
	return &model.CmsPage{
		Identifier:      strPtrIfNotEmpty(p.Identifier),
		URLKey:          strPtrIfNotEmpty(p.Identifier), // url_key == identifier in Magento
		Title:           strPtrIfNotEmpty(p.Title),
		Content:         strPtrIfNotEmpty(p.Content),
		ContentHeading:  strPtrIfNotEmpty(p.ContentHeading),
		PageLayout:      strPtrIfNotEmpty(p.PageLayout),
		MetaTitle:       strPtrIfNotEmpty(p.MetaTitle),
		MetaDescription: strPtrIfNotEmpty(p.MetaDescription),
		MetaKeywords:    strPtrIfNotEmpty(p.MetaKeywords),
		RelativeURL:     &relativeURL,
		RedirectCode:    redirectCode,
		Type:            t,
	}
}

// productRoutable maps a Magento product type_id to the correct Go model stub.
func productRoutable(typeID, relativeURL string, redirectCode int, t *model.URLRewriteEntityTypeEnum) model.RoutableInterface {
	switch typeID {
	case "configurable":
		return &model.ConfigurableProduct{RelativeURL: &relativeURL, RedirectCode: redirectCode, Type: t}
	case "bundle":
		return &model.BundleProduct{RelativeURL: &relativeURL, RedirectCode: redirectCode, Type: t}
	case "virtual":
		return &model.VirtualProduct{RelativeURL: &relativeURL, RedirectCode: redirectCode, Type: t}
	case "downloadable":
		return &model.DownloadableProduct{RelativeURL: &relativeURL, RedirectCode: redirectCode, Type: t}
	case "grouped":
		return &model.GroupedProduct{RelativeURL: &relativeURL, RedirectCode: redirectCode, Type: t}
	default: // "simple" and unknown types
		return &model.SimpleProduct{RelativeURL: &relativeURL, RedirectCode: redirectCode, Type: t}
	}
}

// urlTypeEnum converts a url_rewrite entity_type string to the GraphQL enum pointer.
func urlTypeEnum(entityType string) *model.URLRewriteEntityTypeEnum {
	var t model.URLRewriteEntityTypeEnum
	switch entityType {
	case "product":
		t = model.URLRewriteEntityTypeEnumProduct
	case "category":
		t = model.URLRewriteEntityTypeEnumCategory
	case "cms-page":
		t = model.URLRewriteEntityTypeEnumCmsPage
	default:
		return nil
	}
	return &t
}

// ─── reCAPTCHA ────────────────────────────────────────────────────────────────

// formTypeConfigKey maps the ReCaptchaFormEnum to the Magento config key suffix.
var formTypeConfigKey = map[model.ReCaptchaFormEnum]string{
	model.ReCaptchaFormEnumPlaceOrder:              "place_order",
	model.ReCaptchaFormEnumContact:                 "contact",
	model.ReCaptchaFormEnumCustomerLogin:            "customer_login",
	model.ReCaptchaFormEnumCustomerForgotPassword:   "customer_forgot_password",
	model.ReCaptchaFormEnumCustomerCreate:           "customer_create",
	model.ReCaptchaFormEnumCustomerEdit:             "customer_edit",
	model.ReCaptchaFormEnumNewsletter:               "newsletter",
	model.ReCaptchaFormEnumProductReview:            "product_review",
	model.ReCaptchaFormEnumSendfriend:               "sendfriend",
	model.ReCaptchaFormEnumBraintree:                "braintree",
	model.ReCaptchaFormEnumResendConfirmationEmail:  "resend_confirmation_email",
}

// recaptchaTypeConfigPaths returns the website_key config path for a given recaptcha type string.
func recaptchaWebsiteKeyPath(captchaType string) string {
	return captchaType + "/general/public_key"
}

// recaptchaThemePath returns the theme config path for a captcha type.
func recaptchaThemePath(captchaType string) string {
	return captchaType + "/general/theme"
}

// GetRecaptchaFormConfig returns reCAPTCHA configuration for a specific form.
func (s *StoreService) GetRecaptchaFormConfig(ctx context.Context, formType model.ReCaptchaFormEnum, storeID int) (*model.ReCaptchaConfigOutput, error) {
	configKey, ok := formTypeConfigKey[formType]
	if !ok {
		disabled := false
		return &model.ReCaptchaConfigOutput{IsEnabled: disabled}, nil
	}

	captchaType := s.cp.Get("recaptcha_frontend/type_for/"+configKey, storeID)
	if captchaType == "" {
		disabled := false
		return &model.ReCaptchaConfigOutput{IsEnabled: disabled}, nil
	}

	websiteKey := s.cp.Get(recaptchaWebsiteKeyPath(captchaType), storeID)
	theme := s.cp.Get(recaptchaThemePath(captchaType), storeID)
	if theme == "" {
		theme = "light"
	}
	validationMsg := s.cp.Get("recaptcha_frontend/failure_messages/validation_failure_message", storeID)
	if validationMsg == "" {
		validationMsg = "reCAPTCHA verification failed."
	}
	technicalMsg := s.cp.Get("recaptcha_frontend/failure_messages/technical_failure_message", storeID)
	if technicalMsg == "" {
		technicalMsg = "reCAPTCHA technical problem."
	}
	badgePos := s.cp.Get("recaptcha_frontend/invisible/badge_position", storeID)
	langCode := s.cp.Get("recaptcha_frontend/general/language_code", storeID)
	minScore := s.cp.GetFloat("recaptcha_v3/score/minimum_score", storeID, 0.5)

	rType := recaptchaTypeEnum(captchaType)
	config := &model.ReCaptchaConfiguration{
		ReCaptchaType:            rType,
		WebsiteKey:               websiteKey,
		Theme:                    theme,
		ValidationFailureMessage: validationMsg,
		TechnicalFailureMessage:  technicalMsg,
	}
	if badgePos != "" {
		config.BadgePosition = &badgePos
	}
	if langCode != "" {
		config.LanguageCode = &langCode
	}
	if captchaType == "recaptcha_v3" {
		config.MinimumScore = &minScore
	}

	return &model.ReCaptchaConfigOutput{
		IsEnabled:      true,
		Configurations: config,
	}, nil
}

// GetRecaptchaV3Config returns the reCAPTCHA v3 global configuration.
func (s *StoreService) GetRecaptchaV3Config(ctx context.Context, storeID int) (*model.ReCaptchaConfigurationV3, error) {
	const captchaType = "recaptcha_v3"

	websiteKey := s.cp.Get(recaptchaWebsiteKeyPath(captchaType), storeID)
	theme := s.cp.Get(recaptchaThemePath(captchaType), storeID)
	if theme == "" {
		theme = "light"
	}
	minScore := s.cp.GetFloat("recaptcha_v3/score/minimum_score", storeID, 0.5)
	badgePos := s.cp.Get("recaptcha_frontend/invisible/badge_position", storeID)
	if badgePos == "" {
		badgePos = "bottomright"
	}
	langCode := s.cp.Get("recaptcha_frontend/general/language_code", storeID)
	failureMsg := s.cp.Get("recaptcha_frontend/failure_messages/validation_failure_message", storeID)
	if failureMsg == "" {
		failureMsg = "reCAPTCHA verification failed."
	}

	// Collect which forms have recaptcha_v3 configured
	var forms []model.ReCaptchaFormEnum
	for formEnum, configKey := range formTypeConfigKey {
		if s.cp.Get("recaptcha_frontend/type_for/"+configKey, storeID) == captchaType {
			forms = append(forms, formEnum)
		}
	}

	// isEnabled requires a private key + website key + at least one form
	privateKey := s.cp.Get("recaptcha_v3/general/private_key", storeID)
	isEnabled := privateKey != "" && websiteKey != "" && len(forms) > 0

	result := &model.ReCaptchaConfigurationV3{
		IsEnabled:    isEnabled,
		WebsiteKey:   websiteKey,
		MinimumScore: minScore,
		BadgePosition: badgePos,
		FailureMessage: failureMsg,
		Forms:        forms,
		Theme:        theme,
	}
	if langCode != "" {
		result.LanguageCode = &langCode
	}
	return result, nil
}

// recaptchaTypeEnum maps a Magento captcha type string to the GraphQL enum.
func recaptchaTypeEnum(captchaType string) model.ReCaptchaTypeEnum {
	switch captchaType {
	case "recaptcha_v2_invisible", "recaptcha_v3":
		return model.ReCaptchaTypeEnumInvisible
	case "recaptcha_v2_checkbox":
		return model.ReCaptchaTypeEnumRecaptcha
	default:
		return model.ReCaptchaTypeEnumRecaptchaV3
	}
}

// ─── helpers ──────────────────────────────────────────────────────────────────

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// strPtrIfNotEmpty returns a pointer to s, or nil if s is empty.
// Alias used where strPtr reads ambiguously.
func strPtrIfNotEmpty(s string) *string { return strPtr(s) }

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
