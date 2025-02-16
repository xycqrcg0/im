package controllers

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"im/global"
	"im/models"
	"log"
	"net/http"
)

type requestMsg struct {
	FromID string `json:"from_id"`
	ToID   string `json:"to_id"`
}

func AddFriendRequest(c *gin.Context) {
	data := &requestMsg{}
	if err := c.ShouldBind(&data); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}

	fromUsername := c.MustGet("username")

	targetUser := &models.User{}
	if err := global.DB.Where("user_id=?", data.ToID).First(&targetUser).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "该用户不存在"})
		return
	}

	msg := models.GenerateMessage(data.FromID, data.ToID, 2, fmt.Sprintf("用户%s(%s)请求加你为好友", fromUsername, data.FromID), 0)
	ForwardMessage(msg)
}

func AddFriendResponse(c *gin.Context) {
	response := c.Param("response")
	name := c.MustGet("username").(string)
	id := c.MustGet("userid").(string)
	data := &requestMsg{}
	if err := c.ShouldBind(&data); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}

	if response == "reject" {
		msg := models.GenerateMessage("000000", data.FromID, 2, fmt.Sprintf("用户%s(%s)拒绝了你的请求", name, data.ToID), 2)
		ForwardMessage(msg)
	} else if response == "accept" {
		//获取对方名字
		var friend = &models.User{}
		if err := global.DB.Model(&models.User{}).Where("user_id=?", data.FromID).First(&friend).Error; err != nil {
			log.Println("名字获取失败")
			return
		}
		//把两者关系写入关系表
		var friendship1 = &models.Friendship{
			UserId:     data.ToID,
			FriendId:   data.FromID,
			FriendName: friend.Username,
		}
		var friendship2 = &models.Friendship{
			UserId:     data.FromID,
			FriendId:   data.ToID,
			FriendName: name,
		}
		if err := global.DB.AutoMigrate(&models.Friendship{}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "好友请求中数据库写入出错"})
			return
		}
		//双向
		if err := global.DB.Create(&friendship1).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "好友请求中数据库写入出错"})
			return
		}
		if err := global.DB.Create(&friendship2).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "好友请求中数据库写入出错"})
			return
		}

		//向From用户发送通知
		msg := models.GenerateMessage(data.ToID, data.FromID, 0, "我已同意你的好友请求", 0)
		ForwardMessage(msg)
	}

	conversationId := "000000" + id
	var user = &models.User{}
	if err := global.DB.Where("user_id=?", data.FromID).First(&user).Error; err != nil {
		log.Println("名字获取失败")
		return
	}
	content := fmt.Sprintf("用户%s(%s)请求加你为好友", user.Username, data.FromID)
	log.Println("content", content)
	if err := global.DB.Model(&models.Message{}).Where("conversation_id=?", conversationId).Where("content=?", content).Update("status", 2).Error; err != nil {
		log.Println("系统消息状态修改失败")
	}
}

func DeleteFriend(c *gin.Context) {
	name := c.MustGet("username").(string)
	userid := c.MustGet("userid").(string)
	targetId := c.Param("id")

	//系统消息用户是不能够删除的哦
	if targetId == "000000" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "此用户不支持删除"})
		return
	}

	//删除关系库里的记录
	if err := global.DB.Model(&models.Friendship{}).Where("user_id=? AND friend_id=?", userid, targetId).Delete(&models.Friendship{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "记录删除失败"})
		return
	}
	if err := global.DB.Model(&models.Friendship{}).Where("user_id=? AND friend_id=?", targetId, userid).Delete(&models.Friendship{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "记录删除失败"})
		return
	}
	//清除聊天记录
	conversationId := models.GenerateConversationID(userid, targetId)
	if err := global.DB.Model(&models.Message{}).Where("conversation_id=?", conversationId).Delete(&models.Message{}).Error; err != nil {
		log.Println("聊天记录删除失败")
	}
	//通过系统消息通知被删除好友
	msg := models.GenerateMessage("000000", targetId, 2, fmt.Sprintf("用户%s(%s)删除了与你的好友关系", name, userid), 2)
	ForwardMessage(msg)
}

func GetFriends(c *gin.Context) {
	id := c.MustGet("userid").(string)
	type Friend struct {
		FriendId   string `json:"friend_id"`
		FriendName string `json:"friend_name"`
	}
	var friends []Friend
	if err := global.DB.Model(&models.Friendship{}).Select("friend_id, friend_name").Where("user_id=?", id).Find(&friends).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}
	log.Printf("%v", friends)
	//friendsByte, err := json.Marshal(friends)
	//if err != nil {
	//	log.Println("序列化失败")
	//	return
	//}
	c.JSON(http.StatusOK, friends)
}
