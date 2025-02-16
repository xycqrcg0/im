package models

type Friendship struct {
	UserId     string `json:"user_id"`
	FriendId   string `json:"friend_id"`
	FriendName string `json:"friend_name"`
}
