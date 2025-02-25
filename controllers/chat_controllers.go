package controllers

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"im/global"
	"im/models"
	"im/utils"
	"log"
	"net/http"
	"sync"
	"time"
)

const (
	WriteWait      = 10 * time.Second // 写超时
	PingPeriod     = 30 * time.Second // 心跳间隔
	PongWait       = 60 * time.Second // 等待pong超时
	MaxMessageSize = 1024             // 最大消息大小
)

var (
	websocketUpgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool { //还不清楚
			return true // 先允许所有跨域请求（生产环境需限制）
		},
	}
	onlineUsers sync.Map // 并发安全的在线用户池,键是用户id，值是OnlineUser
	mutex       sync.Mutex
)

type OnlineUser struct { //用来记录在线用户
	UserID    string
	Username  string
	Conn      *websocket.Conn
	LastSeen  time.Time
	CloseChan chan bool
	SendChan  chan []byte
	Heartbeat time.Duration
	Timeout   time.Duration
}

func WebsocketHandler(c *gin.Context) { //对用户进行websocket协议升级,同时加入在线用户池，开启监听循环
	conn, err := websocketUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("error:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"升级失败": err})
		return
	}

	username := c.MustGet("username").(string)
	userid := c.MustGet("userid").(string)
	onlineuser := &OnlineUser{
		Username:  username,
		UserID:    userid,
		Conn:      conn,
		LastSeen:  time.Now(),
		CloseChan: make(chan bool),
		SendChan:  make(chan []byte, 512), // 带缓冲的通道
		Heartbeat: PingPeriod,
		Timeout:   PongWait,
	}

	//用户上线
	onlineUsers.Store(userid, onlineuser)

	defer func() {
		onlineUsers.Delete(userid)
		conn.Close()
	}()

	//启动读协程
	go onlineuser.readPump()

	//感觉在这个时候先把离线消息处理了会比较好，新消息可以先缓存在sendChan里
	//但是问题在于，ping在写协程里，如果redis里信息量太大，读取时间太长，连接可能会因为心跳检测时间太长而断开
	msgs := utils.ReadFromRedis(userid)
	for _, msgByte := range msgs {
		conn.WriteMessage(websocket.TextMessage, []byte(msgByte))
		go func() {
			msg := &models.Message{}
			json.Unmarshal([]byte(msgByte), &msg)
			msg.Status = 1 //已送达
			if err := utils.StoreInMysql(msg); err != nil {
				log.Println("error:", err)
				return
			}
		}()
	}

	//启动写协程
	go onlineuser.writePump()
	<-onlineuser.CloseChan
}

// 读协程，处理消息接收（pong响应会被ponghandler自动处理）
func (u *OnlineUser) readPump() {
	defer func() {
		u.CloseChan <- true
	}()

	u.Conn.SetReadLimit(MaxMessageSize)
	u.Conn.SetReadDeadline(time.Now().Add(u.Timeout))

	u.Conn.SetPongHandler(func(string) error {
		u.Conn.SetReadDeadline(time.Now().Add(u.Timeout))
		u.LastSeen = time.Now()
		log.Printf("已收到用户%s的心跳检测pong", u.UserID)
		return nil
	})

	for {
		//读取消息
		_, msgBytes, err := u.Conn.ReadMessage() //阻塞的，协议层的pong会被自动处理而不被ReadMessage获取
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
				log.Printf("%s异常断开，%v/n", u.UserID, err)
			}
			//此处可以有补充
			u.CloseChan <- true
			break
		}
		message := &models.Message{}
		if err := json.Unmarshal(msgBytes, &message); err != nil {
			log.Printf("消息解析失败，%v", err)
			continue
		}
		message = models.GenerateMessage(message.UserID, message.TargetID, message.Cmd, message.Content, 0)

		//处理业务消息
		if message.Cmd == 1 { //处理群聊消息,鉴于这是个小demo，采用了写扩散（读扩散处理不太会）
			var members []string
			if err := global.DB.Model(&models.GroupMember{}).Where("group_id=?", message.TargetID).Pluck("user_id", &members).Error; err != nil {
				log.Println("群聊成员列表获取失败")
				return
			}
			for _, memberId := range members {
				go ForwardMessage(message, memberId)
			}
			if message.Cmd != 2 {
				message.Status = 1 //状态改为1，来表示消息已经送达
			}
			//将消息存储在历史库里
			if err := utils.StoreInMysql(message); err != nil {
				log.Println("error:", err)
				return
			}
		} else {
			go ForwardMessage(message, "")
		}
	}
}

// 写协程，处理消息发送和ping
func (u *OnlineUser) writePump() {
	ticker := time.NewTicker(u.Heartbeat) //计时器
	defer func() {
		ticker.Stop()
		u.CloseChan <- true
	}()

	for {
		select {
		case message, ok := <-u.SendChan:
			mutex.Lock()
			//设置写超时
			u.Conn.SetWriteDeadline(time.Now().Add(WriteWait))

			if !ok {
				//通道关闭时发送关闭帧通知对端
				u.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := u.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Println("WriteMessage error:", err)
			}
			log.Println("消息发送成功")
			//创建文本消息写入器(另一种方法，不舍得删）
			//w, err := u.Conn.NextWriter(websocket.TextMessage)
			//if err != nil {
			//	return
			//}
			//w.Write(message)
			//if err := w.Close(); err != nil {
			//	return
			//}
			mutex.Unlock()

		case <-ticker.C:
			//发送心跳ping
			u.Conn.SetWriteDeadline(time.Now().Add(WriteWait))
			if err := u.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Print("发送ping失败，连接将退出")
				return
			}
			log.Printf("已向%s发送心跳检测ping", u.UserID)
		}
	}
}

// ForwardMessage 私聊/系统消息时id填""
func ForwardMessage(msg *models.Message, id string) {
	var targetid string
	if id == "" {
		targetid = msg.TargetID
	} else {
		targetid = id
	}

	if targetid == msg.UserID { //群聊中，自己发的信息就不要再收到一遍了（不过这也导致自己不能和自己私聊了）
		return
	}

	msgBytes, _ := json.Marshal(msg)
	target, ok := onlineUsers.Load(targetid)
	if !ok {
		log.Printf("用户%s不在线", targetid)
		//redis离线库
		key := fmt.Sprintf("offline:%s", targetid)
		global.RedisDB.RPush(key, msgBytes)
		return
	}
	targetUser := target.(*OnlineUser)

	targetUser.SendChan <- msgBytes
	//此时消息已经发送到用户的发送通道中，认为消息已经送达，将对消息持久化处理
	//不过这样用户第一次收到的消息结构体里的status都为0，从历史库里再读取时则为1 ，（可以用来区分是不是未读消息？）//所以这个状态会有什么用呢（咳咳）

	if id == "" { //私聊/系统消息时，消息在这里存储；群聊时，为防止同一条消息多次存储，写入历史库独立执行

		if msg.Cmd != 2 {
			msg.Status = 1 //状态改为1，来表示消息已经送达
		}
		//将消息存储在历史库里
		if err := utils.StoreInMysql(msg); err != nil {
			log.Println("error:", err)
			return
		}

		//select {
		//case targetUser.SendChan <- msgBytes:
		//	msg.Status = 1
		//	utils.StoreInMysql(msg)
		//default:
		//	log.Println("发送通道已满")
		//  这种情况还没想好怎么处理比较好，只能先委屈一下用户，先在这里阻塞一下了。（不过用go再开一个线程应该还好）
		//}
	}
}
