package graph

import (
	"database/sql"

	commonconfig "github.com/magendooro/magento2-go-common/config"
	"github.com/magendooro/magento2-store-graphql-go/internal/repository"
	"github.com/magendooro/magento2-store-graphql-go/internal/service"
)

// Resolver is the root resolver. It holds dependencies shared across all resolvers.
type Resolver struct {
	Service *service.StoreService
}

// NewResolver wires all repositories and returns a ready Resolver.
func NewResolver(db *sql.DB) (*Resolver, error) {
	cp, err := commonconfig.NewConfigProvider(db)
	if err != nil {
		return nil, err
	}

	storeRepo := repository.NewStoreRepository(db)
	countryRepo := repository.NewCountryRepository(db)
	currencyRepo := repository.NewCurrencyRepository(db)
	cmsRepo := repository.NewCmsRepository(db)
	urlRepo := repository.NewUrlRepository(db)

	svc := service.NewStoreService(storeRepo, countryRepo, currencyRepo, cmsRepo, urlRepo, cp)

	return &Resolver{Service: svc}, nil
}
