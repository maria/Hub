package model

type User struct {
	ID         int     `bson:"_id"`
	Avatar     string  `bson:"avatar"`
	Username   string  `bson:"username"`
	Fullname   string  `bson:"fullname"`
	Followers  int     `bson:"followers"`
	Following  int     `bson:"following"`
}