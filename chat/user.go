package chat

import (
	"wsl.test/utils"
)

var UserList = map[string]*User{}

type User struct {
	ID   string `json:"id"`
	FQDN string `json:"fqdn"`
}

func CreateUser(fqdn string) *User {
	u := &User{
		FQDN: fqdn,
		ID:   utils.IDGenerator.Generate(),
	}
	UserList[u.ID] = u
	return u
}
