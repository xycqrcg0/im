package controllers

import (
	"github.com/gin-gonic/gin"
	"im/global"
	"im/models"
	"im/utils"
	"log"
	"net/http"
	"regexp"
)

func Register(ctx *gin.Context) {
	var newUser models.User
	if err := ctx.ShouldBind(&newUser); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error1": err.Error()})
		return
	}

	// 正则表达式：只允许字母、数字和_ . ，且长度至少4个字符
	re := regexp.MustCompile("^[a-zA-Z0-9_.]{4,}$")
	if newUser.Username == "" || len(newUser.Username) > 20 || !re.MatchString(newUser.Password) {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "请按要求输入用户名和密码"})
		return
	}
	hashed, err1 := utils.HashPassword(newUser.Password)
	if err1 != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error2": err1.Error()})
		return
	}
	newUser.Password = hashed

	newUser.UserID = utils.GenerateUserID()

	token, err := utils.GenerateJWT(newUser.Username, newUser.UserID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error3": err.Error()})
		return
	}

	if err := global.DB.AutoMigrate(&newUser); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error4": err.Error()})
		return
	}

	if err := global.DB.Create(&newUser).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error5": err.Error()})
		return
	}

	//要先加系统消息的好友哦
	var friendship = &models.Friendship{
		UserId:     newUser.UserID,
		FriendId:   "000000",
		FriendName: "系统消息",
	}
	if err := global.DB.AutoMigrate(&models.Friendship{}); err != nil {
		log.Println("好友添加失败")
		return
	}
	if err := global.DB.Create(&friendship).Error; err != nil {
		log.Println("好友添加失败")
		return
	}

	newUser.Token = token
	ctx.JSON(http.StatusCreated, newUser)
}

func Login(ctx *gin.Context) {
	var input models.User

	if err := ctx.ShouldBind(&input); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error6": err.Error()})
		return
	}

	//系统消息用户id为000000，该账户不能够被登录
	if input.UserID == "000000" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "不支持此id"})
		return
	}

	var user models.User
	if err := global.DB.Where("user_id=?", input.UserID).First(&user).Error; err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error7": err.Error()})
		return
	}

	if err := utils.CheckPassword(user.Password, input.Password); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error8": "账号或密码错误"})
		return
	}

	token, err := utils.GenerateJWT(user.Username, user.UserID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error9": err.Error()})
		return
	}

	user.Token = token
	ctx.JSON(http.StatusOK, user)
}
