package config

import (
	"fmt"
	"github.com/go-redis/redis"
	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"im/global"
	"im/models"
	"log"
	"time"
)

type Config struct {
	App struct {
		Name string
		Port string
	}
	Database struct {
		Dsn          string
		MaxIdleConns int
		MaxOpenConns int
	}
}

var AppConfig *Config

func InitConfig() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./config")
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("配置失败,%s", err)
	}
	AppConfig = &Config{}
	if err := viper.Unmarshal(AppConfig); err != nil {
		log.Fatalf("配置失败，%s", err)
	}

	initDB()
	initRedis()

	//初始化用户id使用表
	global.HaveUsedID = make(map[string]struct{})
	if global.DB.Migrator().HasTable(&models.User{}) {
		var ids []string
		if err := global.DB.Model(&models.User{}).Pluck("user_id", &ids).Error; err != nil {
			log.Fatalf("已注册的id获取失败")
		}
		for _, id := range ids {
			global.HaveUsedID[id] = struct{}{}
		}
	}

	//初始化系统消息用户
	if !global.DB.Migrator().HasTable(&models.User{}) {
		root := &models.User{
			UserID:   "000000",
			Username: "系统消息",
			Password: "114514",
		}
		if err := global.DB.AutoMigrate(&models.User{}); err != nil {
			log.Println("初始用户注册失败")
			return
		}
		if err := global.DB.Create(&root).Error; err != nil {
			log.Println("初始用户注册失败")
			return
		}
	}
}

func initDB() {
	//user := os.Getenv("DB_USER")
	//pwd := os.Getenv("DB_PASSWORD")
	//port := os.Getenv("DB_PORT")
	//name := os.Getenv("DB_NAME")
	port := "3306"
	user := "root"
	pwd := "123456"
	name := "imdb"
	dsn := fmt.Sprintf("%s:%s@tcp(127.0.0.1:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", user, pwd, port, name)
	db, err1 := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err1 != nil {
		log.Fatalf("初始化数据库失败，%s", err1)
	}

	sqlDB, err2 := db.DB()
	if err2 != nil {
		log.Fatalf("初始化数据库失败，%s", err2)
	}

	sqlDB.SetMaxIdleConns(AppConfig.Database.MaxIdleConns)
	sqlDB.SetMaxOpenConns(AppConfig.Database.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Hour)

	global.DB = db
}

func initRedis() {
	global.RedisDB = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		DB:       0,
		Password: "",
	})

	if _, err := global.RedisDB.Ping().Result(); err != nil {
		log.Fatalf("redis连接失败")
	}
}
