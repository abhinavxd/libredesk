package user

import (
	"fmt"
	"strings"
	"sync"
)

var sharedContactMu sync.Mutex

func (u *Manager) SaveSharedContact(firstName, lastName, phone string) (int, error) {
	phone = strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, phone)
	if phone == "" {
		return 0, fmt.Errorf("contact has no phone number")
	}
	sharedContactMu.Lock()
	defer sharedContactMu.Unlock()
	password, err := u.generatePassword()
	if err != nil {
		return 0, err
	}
	var id int
	err = u.q.SaveSharedContact.Get(&id, phone, firstName, lastName, password)
	return id, err
}
