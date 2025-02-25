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
		auth.POST("/register", controllers.Register) //password:只允许字母数字和_ . 且长度至少4个字符；username:不为空且不超过20个字符
		//body里要有json数据{"username":xxx,"password":xxx}
		auth.POST("/login", controllers.Login)
		//body里要有json数据{"userid":xxx,"password":xxx}
	}

	r.GET("/ws", middlewares.AuthMiddleware(), controllers.WebsocketHandler)

	history := r.Group("/api/history")
	history.Use(middlewares.AuthMiddleware())
	{
		history.GET("/:cmd/:target-id", controllers.AcquireHistoryMessages) //hope ?page=***&page-size=*** ,page-size的值不要每次都变
	}

	relation := r.Group("/api/relation")
	relation.Use(middlewares.AuthMiddleware())
	{
		relation.POST("/friend/add", controllers.AddFriendRequest)
		//body里要放着data{FromId string, ToId string}，FromId是请求加好友的一方
		relation.POST("/friend/:response", controllers.AddFriendResponse) //reject or accept
		//body里要放着data{FromId string, ToId string}，FromId是请求加好友的一方
		relation.POST("/friend/delete", controllers.DeleteFriend) //hope ?id=xxx
		relation.GET("/friend/friends", controllers.GetFriends)

		relation.POST("/group/create", controllers.CreateGroup)               //hope ?group-name=xxx
		relation.POST("/group/delete", controllers.DropGroup)                 //hope ?group-id=xxx
		relation.POST("/group/members/push", controllers.PushMember)          //hope ?group-id=xxx&&target-id=xxx
		relation.POST("/group/members/drop", controllers.DropMemberFromGroup) //hope ?group-id=xxx&&target-id=xxx
		relation.POST("/group/add-request", controllers.AddInGroupRequest)    //hope ?group-id=xxx
		relation.POST("/group/add-response", controllers.AddInGroupResponse)
		//body里要放target-id,group-id,
		relation.GET("/group/groups", controllers.GetGroups)

	}

	return r
}
