package router

import (
	"context"
	"reflect"
	"strings"

	http_error "abdanhafidz.com/event-booking-api/models/error"
	"abdanhafidz.com/event-booking-api/provider"
	"abdanhafidz.com/event-booking-api/utils"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

// Version is set at build time (-ldflags "-X .../router.Version=<sha>") so the
// pipeline can tell when a new deployment is live.
var Version = "dev"

// SetupRouter registers every route. Integration tests call it directly and
// serve the engine with httptest.
func SetupRouter(appProvider provider.AppProvider) *gin.Engine {
	router, controller, middleware := appProvider.ProvideRouter(), appProvider.ProvideControllers(), appProvider.ProvideMiddlewares()
	db := appProvider.ProvideConfig().ProvideDatabaseConfig().GetInstance()

	useJSONFieldNames()

	requestMiddleware := middleware.ProvideRequestMiddleware()
	router.Use(requestMiddleware.Log, requestMiddleware.Timeout)

	router.GET("/health", func(ctx *gin.Context) {
		sqlDB, err := db.DB()
		if err == nil {
			err = sqlDB.PingContext(ctx.Request.Context())
		}
		if err != nil {
			ctx.JSON(503, gin.H{"status": "DOWN", "db": "unreachable", "version": Version})
			return
		}
		ctx.JSON(200, gin.H{"status": "OK", "db": "ok", "version": Version})
	})

	EventRouter(router, controller)
	BookingRouter(router, middleware, controller)
	SwaggerRouter(router)

	router.NoRoute(func(ctx *gin.Context) {
		utils.ResponseFAILED[any](ctx, nil, http_error.ROUTE_NOT_FOUND)
	})
	return router
}

func RunRouter(appProvider provider.AppProvider) {
	router := SetupRouter(appProvider)

	bookingConfig := appProvider.ProvideConfig().ProvideBookingConfig()
	if bookingConfig.IsSweeperEnabled() {
		appProvider.ProvideServices().ProvideExpiryService().Start(context.Background(), bookingConfig.GetSweepInterval())
	}

	router.Run(appProvider.ProvideConfig().ProvideEnvConfig().GetTCPAddress())
}

// useJSONFieldNames makes validation errors name the JSON field
// (ticket_type_id) instead of the Go field (TicketTypeId).
func useJSONFieldNames() {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterTagNameFunc(func(f reflect.StructField) string {
			name := strings.SplitN(f.Tag.Get("json"), ",", 2)[0]
			if name == "-" {
				return ""
			}
			return name
		})
	}
}
