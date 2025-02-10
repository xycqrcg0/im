package global

import (
	"github.com/go-redis/redis"
	"gorm.io/gorm"
)

var DB *gorm.DB
var RedisDB *redis.Client

var HaveUsedID map[string]struct{}
