package utils

import (
	"fmt"
	"im/global"
	"im/models"
	"log"
)

// StoreInMysql 将消息存入历史库
func StoreInMysql(msg *models.Message) {
	if err := global.DB.AutoMigrate(msg); err != nil {
		log.Println("模型绑定失败")
		return
	}
	if err := global.DB.Create(msg).Error; err != nil {
		log.Println("消息存储失败")
		return
	}
	return
}

// ReadFromRedis 读取redis离线库里的消息并存入历史库（用户刚上线时应调用一次）
func ReadFromRedis(userid string) []string {
	key := fmt.Sprintf("offline:%s", userid)
	offlineMsgs := global.RedisDB.LRange(key, 0, -1)
	msgs, err := offlineMsgs.Result()
	if err != nil {
		log.Println("离线消息获取失败")
		return nil
	}
	global.RedisDB.Del(key) //清除缓存
	return msgs
}
