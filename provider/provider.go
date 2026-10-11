package provider

import (
	"log"

	entity "abdanhafidz.com/event-booking-api/models/entity"
	"github.com/gin-gonic/gin"
)

type AppProvider interface {
	ProvideRouter() *gin.Engine
	ProvideConfig() ConfigProvider
	ProvideRepositories() RepositoriesProvider
	ProvideServices() ServicesProvider
	ProvideControllers() ControllerProvider
	ProvideMiddlewares() MiddlewareProvider
}
type appProvider struct {
	ginRouter            *gin.Engine
	configProvider       ConfigProvider
	repositoriesProvider RepositoriesProvider
	servicesProvider     ServicesProvider
	controllerProvider   ControllerProvider
	middlewareProvider   MiddlewareProvider
}

func NewAppProvider() AppProvider {
	log.Println("[BOOT] Initializing Config Provider")
	return NewAppProviderWith(NewConfigProvider())
}

func NewAppProviderWith(configProvider ConfigProvider) AppProvider {
	log.Println("[BOOT] Initializing App Provider...")

	// gin.New instead of gin.Default: the request middleware does the logging
	// as one JSON line per request (NFR-05).
	ginRouter := gin.New()
	ginRouter.Use(gin.Recovery())

	repositoriesProvider := NewRepositoriesProvider(configProvider)
	servicesProvider := NewServicesProvider(repositoriesProvider, configProvider)
	controllerProvider := NewControllerProvider(servicesProvider)
	middlewareProvider := NewMiddlewareProvider(repositoriesProvider, configProvider)

	log.Println("[BOOT][DB] Starting database migration...")
	if err := Migrate(configProvider); err != nil {
		log.Fatalf("[BOOT][DB] ❌ Database migration failed: %v", err)
	}
	log.Println("[BOOT][DB] ✅ Database migration completed")

	return &appProvider{
		ginRouter:            ginRouter,
		configProvider:       configProvider,
		repositoriesProvider: repositoriesProvider,
		servicesProvider:     servicesProvider,
		controllerProvider:   controllerProvider,
		middlewareProvider:   middlewareProvider,
	}
}

func Migrate(configProvider ConfigProvider) error {
	return configProvider.ProvideDatabaseConfig().AutoMigrateAll(entity.AllEntities()...)
}

func (a *appProvider) ProvideRouter() *gin.Engine {
	return a.ginRouter
}
func (a *appProvider) ProvideConfig() ConfigProvider {
	return a.configProvider
}

func (a *appProvider) ProvideRepositories() RepositoriesProvider {
	return a.repositoriesProvider
}

func (a *appProvider) ProvideServices() ServicesProvider {
	return a.servicesProvider
}

func (a *appProvider) ProvideControllers() ControllerProvider {
	return a.controllerProvider
}

func (a *appProvider) ProvideMiddlewares() MiddlewareProvider {
	return a.middlewareProvider
}
