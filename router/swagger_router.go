package router

import (
	_ "abdanhafidz.com/event-booking-api/swagger/docs"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// SwaggerRouter serves the UI at /swagger/index.html and the OpenAPI document
// at /swagger/doc.json, which the CI smoke test reads.
func SwaggerRouter(router *gin.Engine) {
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}
