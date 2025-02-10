package models

import (
	"github.com/sony/sonyflake"
	"strconv"
)

type Message struct {
	Id           uint64 `json:"id"`            //消息ID
	ConversionID string `json:"conversion_id"` //会话ID
	UserID       string `json:"user_id"`       //谁发的
	TargetID     string `json:"target_id"`     //对端用户ID/群ID
	Cmd          int    `json:"cmd"`           // 0=私聊 1=群聊 2=系统通知
	Content      string `json:"content"`       //消息的内容
	Status       int    `json:"status"`        //0表示未收到，1表示已收到；系统消息中，2表示消息已经处理
}

// 雪花算法生成与时间有关的有序唯一id
var flake = sonyflake.NewSonyflake(sonyflake.Settings{})

func GenerateMessage(userId string, targetId string, cmd int, content string, status int) *Message {
	id, _ := flake.NextID()
	var msg = &Message{
		Id:       id,
		UserID:   userId,
		TargetID: targetId,
		Cmd:      cmd,
		Content:  content,
		Status:   status,
	}
	//单聊时会话id是两人id组合，群聊时是群id，系统消息是000000+用户id
	var conversionID string
	switch cmd {
	case 0:
		conversionID = GenerateConversionID(userId, targetId)
	case 1:
		conversionID = targetId
	case 2:
		conversionID = "000000" + targetId
	}
	msg.ConversionID = conversionID
	return msg
}

func GenerateConversionID(id1 string, id2 string) string {
	partA, _ := strconv.Atoi(id1)
	partB, _ := strconv.Atoi(id2)
	if partA < partB {
		return id1 + id2
	} else {
		return id2 + id1
	}
}
