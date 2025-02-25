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

	//提供默认值
	if page == 0 {
		page = 1
	}
	if pageSize == 0 {
		pageSize = 20
	}

	offset := pageSize * (page - 1)
	messages := make([]models.Message, 0)
	var conversationId string
	switch cmd {
	//单聊
	case "0":
		//结合用户本身id得到对话id
		conversationId = models.GenerateConversationID(userid, targetId)
	//群聊
	case "1":
		//对话id就是群id
		conversationId = targetId
		//这里要检验一下此用户是不是群聊成员
		var count int64
		if err := global.DB.Model(&models.GroupMember{}).Where("group_id=? AND user_id=?", targetId, userid).Count(&count).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err})
			return
		}
		if count == 0 {
			//检查一下群聊是否存在
			var co int64
			if err := global.DB.Model(&models.Group{}).Where("id=?", targetId).Count(&co).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err})
				return
			}
			if co == 0 {
				c.JSON(http.StatusNotFound, gin.H{"error": "群聊不存在"})
				return
			}
			c.JSON(http.StatusBadRequest, gin.H{"error": "没有权限"})
			return
		}
	//系统消息
	case "2":
		conversationId = "000000" + userid
	}

	if err := global.DB.Model(&models.Message{}).Order("timestamp DESC").Where("conversation_id=?", conversationId).Limit(pageSize).Offset(offset).Find(&messages).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "数据读取失败"})
		return
	}

	c.JSON(http.StatusOK, messages)
}
