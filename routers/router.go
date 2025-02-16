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

	history := r.Group("/api/history")
	history.Use(middlewares.AuthMiddleware())
	{
		history.GET("/:cmd/:target-id", controllers.AcquireHistoryMessages) //( ?page=***&page-size=***),page-size的值不要每次都变
	}

	relation := r.Group("/api/relation")
	relation.Use(middlewares.AuthMiddleware())
	{
		relation.POST("/friend-add", controllers.AddFriendRequest)
		relation.POST("/friend-response/:response", controllers.AddFriendResponse)
		relation.POST("/friend-delete/:id", controllers.DeleteFriend)
		relation.GET("/friends", controllers.GetFriends)
	}

	return r
}
