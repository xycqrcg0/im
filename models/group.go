package models

import (
	"gorm.io/gorm"
	"time"
)

type Group struct {
	Id        string `gorm:"primarykey"`
	OwnerId   string `gorm:"owner_id"`
	GroupName string
	CreatedAt time.Time
}

type GroupMember struct {
	gorm.Model
	GroupName string `gorm:"group_name"`
	GroupId   string `gorm:"group_id"`
	UserId    string `gorm:"user_id"`
}
