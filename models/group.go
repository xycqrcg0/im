package models

import "gorm.io/gorm"

type Group struct {
	gorm.Model
	Id        string
	OwnerId   string
	GroupName string
}

type GroupMember struct {
	gorm.Model
	GroupId string
	UserId  string
}
