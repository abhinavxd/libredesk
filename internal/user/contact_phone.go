package user

import (
	"fmt"
	"sync"

	"github.com/abhinavxd/libredesk/internal/stringutil"
)

var sharedContactMu sync.Mutex

func (u *Manager) SaveSharedContact(firstName, lastName, phone string) (int, error) {
	phone = stringutil.NormalizeWhatsAppPhone(phone)
	if phone == "" {
		return 0, fmt.Errorf("contact has no phone number")
	}
	sharedContactMu.Lock()
	defer sharedContactMu.Unlock()
	password, err := u.newContactPassword()
	if err != nil {
		return 0, err
	}
	var id int
	err = u.q.SaveSharedContact.Get(&id, phone, firstName, lastName, password)
	return id, err
}
