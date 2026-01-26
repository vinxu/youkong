package validator

import (
	"regexp"
	"unicode/utf8"
)

var phoneRegex = regexp.MustCompile(`^1[3-9]\d{9}$`)

func IsValidPhone(phone string) bool {
	return phoneRegex.MatchString(phone)
}

func IsValidSMSCode(code string) bool {
	if len(code) != 6 {
		return false
	}
	for _, c := range code {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func IsValidNickname(nickname string) bool {
	length := utf8.RuneCountInString(nickname)
	return length >= 1 && length <= 20
}

func IsValidCircleName(name string) bool {
	length := utf8.RuneCountInString(name)
	return length >= 1 && length <= 10
}

func IsValidEmoji(emoji string) bool {
	length := utf8.RuneCountInString(emoji)
	return length >= 1 && length <= 2
}

func IsValidColor(color string) bool {
	if len(color) != 7 {
		return false
	}
	if color[0] != '#' {
		return false
	}
	for i := 1; i < 7; i++ {
		c := color[i]
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}
