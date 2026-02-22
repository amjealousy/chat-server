package chat

import (
	"wsl.test/utils"
)

var GlobalUserList = map[string]*User{}

type User struct {
	ID   string `json:"id"`
	FQDN string `json:"fqdn"`
}

func CreateUser(fqdn string) *User {
	u := &User{
		FQDN: fqdn,
		ID:   utils.IDGenerator.Generate(),
	}
	GlobalUserList[u.ID] = u
	return u
}
