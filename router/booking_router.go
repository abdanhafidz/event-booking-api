package router

import (
	"abdanhafidz.com/event-booking-api/provider"
	"github.com/gin-gonic/gin"
)

func BookingRouter(router *gin.Engine, middleware provider.MiddlewareProvider, controller provider.ControllerProvider) {
	bookingController := controller.ProvideBookingController()
	routerGroup := router.Group("/v1/bookings")
	routerGroup.Use(middleware.ProvideUserMiddleware().RequireUser)
	{
		routerGroup.POST("", bookingController.Create)
		routerGroup.GET("/:id", bookingController.Get)
		routerGroup.POST("/:id/pay", bookingController.Pay)
		routerGroup.POST("/:id/cancel", bookingController.Cancel)
	}
}
