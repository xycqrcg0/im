package controllers

import (
	"github.com/gin-gonic/gin"
	"im/global"
	"im/models"
	"net/http"
	"strconv"
)

func AcquireHistoryMessages(c *gin.Context) {
	//ps:此处可以通过redis缓存策略缓解数据库压力，but还不太会，先暂放
	userid := c.MustGet("userid").(string)
	cmd := c.Param("cmd")
	targetId := c.Param("target-id")
	page, _ := strconv.Atoi(c.Query("page"))
	pageSize, _ := strconv.Atoi(c.Query("page-size")) //这个值希望前端传的值是固定的
	offset := pageSize * (page - 1)
	messages := make([]models.Message, 0)
	var conversationId string
	switch cmd {
	//单聊
	case "0":
		conversationId = models.GenerateConversationID(userid, targetId)
	//群聊
	case "1":
		conversationId = targetId
	//系统消息
	case "2":
		conversationId = "000000" + userid
	}

	if err := global.DB.Model(&models.Message{}).Order("timestamp DESC").Where("conversation_id=?", conversationId).Limit(pageSize).Offset(offset).Find(&messages).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "数据读取失败"})
		return
	}

	c.JSON(http.StatusOK, messages)
}
