package utils

import (
	"fmt"
	"im/global"
	"im/models"
	"log"
)

// StoreInMysql 将消息存入历史库
func StoreInMysql(msg *models.Message) error {
	if err := global.DB.AutoMigrate(&models.Message{}); err != nil {
		log.Println("数据库绑定失败")
		return err
	}
	if err := global.DB.Model(&models.Message{}).Create(msg).Error; err != nil {
		log.Println("消息存储失败")
		return err
	}
	return nil
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
