package router

import (
	"abdanhafidz.com/event-booking-api/provider"
	"github.com/gin-gonic/gin"
)

func EventRouter(router *gin.Engine, controller provider.ControllerProvider) {
	eventController := controller.ProvideEventController()
	routerGroup := router.Group("/v1/events")
	{
		routerGroup.GET("/:id", eventController.Get)
	}
}
