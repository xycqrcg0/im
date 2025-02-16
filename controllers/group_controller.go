package controllers

import (
	"github.com/gin-gonic/gin"
	"im/global"
	"im/models"
	"im/utils"
	"log"
	"net/http"
)

func PushMemberInGroup(groupId string, userid string) {
	if err := global.DB.AutoMigrate(&models.GroupMember{}); err != nil {
		log.Println("groupMember模型绑定失败")
		return
	}
	member := &models.GroupMember{
		GroupId: groupId,
		UserId:  userid,
	}
	if err := global.DB.Create(&member).Error; err != nil {
		log.Println("群组成员写入数据库失败")
		return
	}
}

func CreateGroup(c *gin.Context) {
	userid := c.MustGet("userid").(string)
	groupName := c.Param("group-name")
	id := utils.GenerateUserID() //借用一下生成用户id的方法，即群聊与用户共享一套id生成规则(简化)
	newGroup := &models.Group{
		Id:        id,
		OwnerId:   userid,
		GroupName: groupName,
	}
	if err := global.DB.AutoMigrate(&models.Group{}); err != nil {
		log.Println("群组模型绑定失败")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}
	if err := global.DB.Create(&newGroup).Error; err != nil {
		log.Println("群组信息写入数据库失败")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}
	PushMemberInGroup(id, userid)
	c.JSON(http.StatusOK, newGroup)
}
