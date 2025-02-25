package controllers

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"im/global"
	"im/models"
	"im/utils"
	"log"
	"net/http"
)

// PushMemberInGroup 辅助函数，加好友入群
func PushMemberInGroup(groupName string, groupId string, userid string) error {
	if err := global.DB.AutoMigrate(&models.GroupMember{}); err != nil {
		log.Println("groupMember模型绑定失败")
		return err
	}
	member := &models.GroupMember{
		GroupName: groupName,
		GroupId:   groupId,
		UserId:    userid,
	}

	//检验是否重复添加
	var c int64
	if err := global.DB.Model(&models.GroupMember{}).Where("group_id=? AND user_id=?", groupId, userid).Count(&c).Error; err != nil {
		log.Println("pushMemberInGroup出错")
		return err
	}
	if c != 0 {
		return errors.New("请勿重复添加")
	}
	//添加进成员库
	if err := global.DB.Create(&member).Error; err != nil {
		log.Println("群组成员写入数据库失败")
		return err
	}
	return nil
}

// GetGroupMembers 辅助函数，获取群成员列表
func GetGroupMembers(groupId string) ([]string, error) {
	memberIds := make([]string, 0)
	if err := global.DB.Model(&models.GroupMember{}).Where("group_id=?", groupId).Pluck("user_id", &memberIds).Error; err != nil {
		log.Println("获取群成员列表失败")
		return nil, err
	}
	return memberIds, nil
}

// CreateGroup 创建群聊
func CreateGroup(c *gin.Context) {
	userid := c.MustGet("userid").(string)
	groupName := c.Query("group-name")

	//群聊名称不能为空且长度不超过20字符
	if groupName == "" || len(groupName) > 20 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请正确输入群聊名称"})
		return
	}

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
	if err := PushMemberInGroup(groupName, id, userid); err != nil {
		if errors.Is(err, errors.New("请勿重复添加")) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请勿重复添加"})
			return
		} else {
			log.Println("pushMemberInGroup使用出现问题")
			c.JSON(http.StatusInternalServerError, gin.H{"error": err})
			return
		}
	}
	c.JSON(http.StatusOK, newGroup)
}

// DropGroup 解散群聊
func DropGroup(c *gin.Context) {
	//只有群主才可以执行
	//想法：是POST，群聊id用query传递
	userId := c.MustGet("userid").(string)
	groupId := c.Query("group-id")

	//得提供groupId吧
	if groupId == "" {
		c.JSON(http.StatusNotFound, nil)
		return
	}

	var targetGroup = &models.Group{}
	if err := global.DB.Model(&models.Group{}).Where("id=?", groupId).First(&targetGroup).Error; err != nil {
		//groupId得合理吧
		c.JSON(http.StatusNotFound, gin.H{"error": err})
		return
	}

	if targetGroup.OwnerId != userId {
		c.JSON(http.StatusForbidden, gin.H{"error": "抱歉，您没有权限"})
		return
	}

	//将group与group的message和member人员均删除
	if err := global.DB.Model(&models.Group{}).Where("id=?", groupId).Delete(&models.Group{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}
	if err := global.DB.Model(&models.Message{}).Where("conversation_id=?", groupId).Delete(&models.Message{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}
	//删除群成员前先获取一下名单
	memberIds := make([]string, 0)
	var err error
	if memberIds, err = GetGroupMembers(groupId); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}
	//删除
	if err := global.DB.Model(&models.GroupMember{}).Where("group_id=?", groupId).Delete(&models.GroupMember{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}

	//向所有群成员发送群聊消息已解散的消息
	text := fmt.Sprintf("通知：群主已经解散该群聊：%s(%s)", targetGroup.GroupName, targetGroup.Id)
	for _, memberId := range memberIds {
		msg := models.GenerateMessage("000000", memberId, 2, text, 0)
		ForwardMessage(msg, "")
	}

	c.JSON(http.StatusOK, "")
}

// PushMember 邀请群成员,只有群主可以邀请
func PushMember(c *gin.Context) {
	userId := c.MustGet("userid").(string)
	groupId := c.Query("group-id")
	targetId := c.Query("target-id")

	if groupId == "" || targetId == "" {
		c.JSON(http.StatusNotFound, nil)
		return
	}

	var targetGroup = &models.Group{}
	if err := global.DB.Model(&models.Group{}).Where("id=?", groupId).First(&targetGroup).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err})
		return
	}
	if userId != targetGroup.OwnerId {
		c.JSON(http.StatusForbidden, gin.H{"error": "抱歉，您没有权限"})
		return
	}

	//这个新member不能是随随便便的人，要是群主的好友
	var ju int64
	if err := global.DB.Model(&models.Friendship{}).Where("user_id=? AND friend_id=?", userId, targetId).Count(&ju).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}
	if ju == 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "没有权限邀请陌生人"})
		return
	}

	//把targetId放进member表里
	if err := PushMemberInGroup(targetGroup.GroupName, groupId, targetId); err != nil {
		if errors.Is(err, errors.New("请勿重复添加")) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请勿重复添加"})
			return
		} else {
			log.Println("pushMemberInGroup使用出现问题")
			c.JSON(http.StatusInternalServerError, gin.H{"error": err})
			return
		}
	}
	//告诉目标用户该信息
	text := fmt.Sprintf("通知：群主(%s)将您拉入了群聊：%s(%s)", targetGroup.OwnerId, targetGroup.GroupName, targetGroup.Id)
	msg := models.GenerateMessage("000000", targetId, 2, text, 2)
	ForwardMessage(msg, "")
	//目标用户还要更新一下群聊列表

	//通知群成员
	//发送在群聊里的系统消息
	//先获取新成员名字:用id查表
	var targetName string
	if err := global.DB.Model(&models.User{}).Where("user_id=?", targetId).Pluck("username", &targetName).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}
	content := fmt.Sprintf("用户%s(%s)加入了群聊", targetName, targetId)
	sysMsg := models.GenerateMessage("000000", groupId, 1, content, 0)
	memberIds, err := GetGroupMembers(groupId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}
	for _, member := range memberIds {
		go ForwardMessage(sysMsg, member)
	}

	if err := utils.StoreInMysql(sysMsg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}

	c.JSON(http.StatusOK, "")
}

// DropMemberFromGroup 删除群成员
func DropMemberFromGroup(c *gin.Context) {
	userid := c.MustGet("userid").(string)
	groupId := c.Query("group-id")
	targetId := c.Query("target-id")

	if groupId == "" || targetId == "" {
		c.JSON(http.StatusNotFound, nil)
		return
	}

	group := &models.Group{}
	if err := global.DB.Model(&models.Group{}).Where("id=?", groupId).First(&group).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err})
		return
	}
	//只能群主踢人
	if targetId != userid && userid != group.OwnerId {
		c.JSON(http.StatusForbidden, gin.H{"error": "没有权限"})
		return
	}
	//群主不能退群
	if targetId == userid && userid == group.OwnerId {
		c.JSON(http.StatusBadRequest, gin.H{"error": "群主不能退群"})
		return
	}
	//只能踢群里的人
	var ju int64
	if err := global.DB.Model(&models.GroupMember{}).Where("group_id=? AND user_id=?", groupId, targetId).Count(&ju).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "群无此人"})
		return
	}

	//删除用户member表里的信息
	if err := global.DB.Model(&models.GroupMember{}).Where("group_id=? AND user_id=?", groupId, targetId).Delete(&models.GroupMember{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
	}

	//这里再通知一下member
	text := fmt.Sprintf("通知：群主已经将您移出该群聊：%s(%s)", group.GroupName, group.Id)
	msg := models.GenerateMessage("000000", targetId, 2, text, 0)
	ForwardMessage(msg, "")

	c.JSON(http.StatusOK, "")
}

// AddInGroupRequest 申请加入群聊
func AddInGroupRequest(c *gin.Context) {
	username := c.MustGet("username")
	userid := c.MustGet("userid")
	targetGroupID := c.Query("group-id")

	if targetGroupID == "" {
		c.JSON(http.StatusNotFound, nil)
		return
	}

	//已经是群成员就不要再玩了
	var ju int64
	if err := global.DB.Model(&models.GroupMember{}).Where("group_id=? AND user_id=?", targetGroupID, userid).Count(&ju).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}
	if ju != 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "请勿重复进群"})
		return
	}

	//生成系统消息发送给群主
	group := &models.Group{}
	if err := global.DB.Model(&models.Group{}).Where("id=?", targetGroupID).First(&group).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}
	content := fmt.Sprintf("用户%s(%s)申请加入群聊%s(%s)", username, userid, group.GroupName, group.Id)
	msg := models.GenerateMessage("000000", group.OwnerId, 2, content, 0)
	ForwardMessage(msg, "")

	c.JSON(http.StatusOK, nil)
}

// AddInGroupResponse 回复申请加群请求
func AddInGroupResponse(c *gin.Context) {
	userid := c.MustGet("userid").(string)
	type temp struct {
		TargetId string `json:"target-id"`
		GroupId  string `json:"group-id"`
	}
	data := &temp{}
	if err := c.ShouldBind(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}
	targetId := data.TargetId //前端应该是可以得知用户id的，系统消息的content里有(应该好提取出来吧)
	groupId := data.GroupId   //前端应该是可以得知群id的，系统消息的content里有(应该好提取出来吧)

	//先检验一下当前用户是不是群主
	var group = &models.Group{}
	if err := global.DB.Model(&models.Group{}).Where("id=?", groupId).First(&group).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err})
		return
	}
	if group.OwnerId != userid {
		c.JSON(http.StatusForbidden, gin.H{"error": "您没有权限"})
		return
	}

	//获取一下用户名和群聊名
	groupName := group.GroupName

	var targetName string
	if err := global.DB.Model(&models.User{}).Where("user_id=?", targetId).Pluck("username", &targetName).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户id有误"})
		return
	}
	//生成content
	content := fmt.Sprintf("用户%s(%s)申请加入群聊%s(%s)", targetName, targetId, groupName, groupId)

	//这个人得是确实向群主发送过申请的
	var ju int64
	if err := global.DB.Model(&models.Message{}).Where("conversation_id=? AND content=?", "000000"+userid, content).Count(&ju).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}
	if ju == 0 {
		c.JSON(http.StatusNotFound, nil)
		return
	}

	//把人拉进来
	if err := PushMemberInGroup(group.GroupName, group.Id, targetId); err != nil {
		if errors.Is(err, errors.New("请勿重复添加")) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err})
			return
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err})
			return
		}
	}

	//要啰嗦了，从PushMember里copy来一部分

	//告诉目标用户该信息
	text := fmt.Sprintf("通知：群主(%s)将您拉入了群聊：%s(%s)", group.OwnerId, group.GroupName, group.Id)
	msg := models.GenerateMessage("000000", targetId, 2, text, 2)
	ForwardMessage(msg, "")
	//目标用户还要更新一下群聊列表

	//通知群成员
	//发送在群聊里的系统消息

	content2 := fmt.Sprintf("用户%s(%s)加入了群聊", targetName, targetId)
	sysMsg := models.GenerateMessage("000000", groupId, 1, content2, 0)
	memberIds, err := GetGroupMembers(groupId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}
	for _, member := range memberIds {
		go ForwardMessage(sysMsg, member)
	}

	if err := utils.StoreInMysql(sysMsg); err != nil {
		log.Println("a")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}

	//修改群主那里入群申请的系统消息状态
	if err := global.DB.Model(&models.Message{}).Where("conversation_id=? AND content=?", "000000"+userid, content).Update("status", 2).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "系统消息状态修改失败"})
		return
	}

	c.JSON(http.StatusOK, "")
}

// GetGroups 查找列表里的群聊
func GetGroups(c *gin.Context) {
	//从groupMember里查
	userid := c.MustGet("userid")
	type temp struct {
		GroupName string `gorm:"group_name"`
		GroupId   string `gorm:"group_id"`
	}
	groups := make([]temp, 0)
	if err := global.DB.Model(&models.GroupMember{}).Where("user_id=?", userid).Find(&groups).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err})
		return
	}

	c.JSON(http.StatusOK, groups) //ps:c.JSON()会自动进行序列化
}
