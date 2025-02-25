package models

import (
	"github.com/sony/sonyflake"
	"strconv"
	"time"
)

type Message struct { //这个结构体是用来存储消息的
	Id             uint64 `json:"id" gorm:"primarykey"` //消息ID //此项无需客户端填写，传json时填0即可
	ConversationID string `json:"conversation_id"`      //会话ID //此项无需客户端填写，传json时填""即可
	UserID         string `json:"user_id"`              //谁发的
	TargetID       string `json:"target_id"`            //对端用户ID/群ID
	Cmd            int    `json:"cmd"`                  // 0=私聊 1=群聊 2=系统通知
	Content        string `json:"content"`              //消息的内容
	Status         int    `json:"status"`               //0表示未收到，1表示已收到；系统消息中，2表示消息已经处理 //此项无需客户端填写，传json时填0即可
	Timestamp      int64  `json:"timestamp"`            //毫秒级时间戳 //此项无需客户端填写，传json时填0即可
}

// 雪花算法生成与时间有关的有序唯一id
var flake = sonyflake.NewSonyflake(sonyflake.Settings{})

func GenerateMessage(userId string, targetId string, cmd int, content string, status int) *Message {
	id, _ := flake.NextID()
	//单聊时会话id是两人id组合，群聊时是群id，系统消息是000000+用户id
	var conversationID string
	switch cmd {
	case 0:
		conversationID = GenerateConversationID(userId, targetId)
	case 1:
		conversationID = targetId
	case 2:
		conversationID = "000000" + targetId
	}

	var msg = &Message{
		Id:             id,
		ConversationID: conversationID,
		UserID:         userId,
		TargetID:       targetId,
		Cmd:            cmd,
		Content:        content,
		Status:         status,
		Timestamp:      time.Now().UnixNano(),
	}
	return msg
}

func GenerateConversationID(id1 string, id2 string) string {
	partA, _ := strconv.Atoi(id1)
	partB, _ := strconv.Atoi(id2)
	if partA < partB {
		return id1 + id2
	} else {
		return id2 + id1
	}
}

/*
message分类：
userid 正常， cmd = 0 : 正常私聊，对应conversation_id = user_id组合
userid 正常， cmd = 1 : 正常群聊，对应conversation_id = group_id
userid 000000， cmd = 2 : 正常系统消息，好友请求与群聊外部通知，如通知某人自己进入了每个群聊，对应conversation_id = 000000 + user_id
userid 000000， cmd = 1 : 群聊内部通知消息，如xxx加入了群聊，对应conversation_id = group_id
*/
