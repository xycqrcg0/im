package middlewares

import (
	"github.com/gin-gonic/gin"
	"im/utils"
	"log"
	"net/http"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		log.Println("token:", token)
		if token == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error1": "请求头不存在"})
			c.Abort()
			return
		}
		name, id, err := utils.ParseJWT(token)
		if err != nil {
			log.Println("error2:", err)
			c.JSON(http.StatusBadRequest, gin.H{"error2": err})
			c.Abort()
			return
		}
		//上面有亿点啰嗦~
		c.Set("username", name)
		c.Set("userid", id)
		c.Next()
	}
}
