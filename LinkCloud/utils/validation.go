package utils

import "unicode/utf8"

func IsValidUserNameLength(userName string) bool {
	length := utf8.RuneCountInString(userName)
	return length >= 3 && length <= 20
}

func IsValidPasswordLength(password string) bool {
	length := utf8.RuneCountInString(password)
	return length >= 6 && length <= 20
}
