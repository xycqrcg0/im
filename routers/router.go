package routers

import (
	"github.com/gin-gonic/gin"
	"im/controllers"
	"im/middlewares"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()

	auth := r.Group("/api/auth")
	{
		auth.POST("/register", controllers.Register)
		auth.POST("/login", controllers.Login)
	}

	r.GET("/ws", middlewares.AuthMiddleware(), controllers.WebsocketHandler)

	relation := r.Group("/api/relation")
	relation.Use(middlewares.AuthMiddleware())
	{
		relation.POST("/friend-add", controllers.AddFriendRequest)
		relation.POST("/friend-response/:response", controllers.AddFriendResponse)
		relation.POST("/friend-delete", controllers.DeleteFriend)
	}
	return r
}
