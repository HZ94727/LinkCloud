package controller

import "unicode/utf8"

func isValidUserNameLength(userName string) bool {
	length := utf8.RuneCountInString(userName)
	return length >= 3 && length <= 20
}
