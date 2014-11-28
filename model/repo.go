package model

type Repo struct {
	ID           int     `bson:"_id"`
	Name         string  `bson:"name"`
	User         string  `bson:"user"`
	Owner        string  `bson:"owner"`
	Fork         bool    `bson:"fork"`
	Description  string  `bson:"description"`
	Stars        int     `bson:"stars"`
	Forks        int     `bson:"forks"`
	Watchers     int     `bson:"watches"`
}