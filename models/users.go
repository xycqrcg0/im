package models

import "gorm.io/gorm"

type User struct {
	gorm.Model
	Username string `gorm:"not null" json:"username"`
	Password string `gorm:"not null" json:"password"`
	Token    string `json:"token"`
	UserID   string `gorm:"unique;not null" json:"userid"`
}
