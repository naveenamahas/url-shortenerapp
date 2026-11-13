// Package utils holds small helpers: input validation and short code generation.
package utils

import (
	"errors"
	"net/mail"
	"net/url"
	"regexp"
	"strings"
)

const MaxURLLength = 2048

var aliasRe = regexp.MustCompile(`^[a-zA-Z0-9_-]{3,32}$`)

// Words that would clash with our own routes or look official.
var reserved = map[string]bool{"api": true, "health": true, "admin": true, "login": true, "signup": true}

// ValidateURL checks length, scheme (http/https only), host, and that the URL
// does not point back at this service (which would create a redirect loop).
func ValidateURL(raw, selfHost string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", errors.New("url is required")
	}
	if len(raw) > MaxURLLength {
		return "", errors.New("url is too long (max 2048 characters)")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", errors.New("url is not valid")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", errors.New("url must start with http:// or https://")
	}
	host := u.Hostname()
	if host == "" || (!strings.Contains(host, ".") && host != "localhost" && !strings.Contains(host, ":")) {
		return "", errors.New("url has an invalid host")
	}
	if u.User != nil {
		return "", errors.New("url must not contain credentials")
	}
	if strings.EqualFold(u.Host, selfHost) {
		return "", errors.New("cannot shorten a link that points to this service")
	}
	return u.String(), nil
}

func ValidateAlias(alias string) error {
	if !aliasRe.MatchString(alias) {
		return errors.New("alias must be 3-32 characters: letters, numbers, - or _")
	}
	if reserved[strings.ToLower(alias)] {
		return errors.New("this alias is reserved")
	}
	return nil
}

func ValidateEmail(email string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email || len(email) > 254 {
		return "", errors.New("email is not valid")
	}
	return email, nil
}
