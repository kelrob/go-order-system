package email

import "net/mail"

func IsValid(email string) bool {
	addr, err := mail.ParseAddress(email)
	return err == nil && addr.Address == email
}
