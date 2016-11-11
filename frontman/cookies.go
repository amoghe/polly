package main

import (
	"net/http"

	"github.com/dghubble/gologin"
)

var (
	sessionCookieMaker = CookieMaker{
		Name:     "session-cookie",
		Path:     "/",
		MaxAge:   3600, // FIXME
		HTTPOnly: true,
		Secure:   false, // FIXME
	}

	stateCookieMaker = CookieMaker{
		Name:     "state-cookie",
		Path:     "/",
		MaxAge:   60,
		HTTPOnly: true,
		Secure:   false, // FIXME
	}
)

// CookieMaker creates new cookies
type CookieMaker gologin.CookieConfig

// NewCookie returns a new http.Cookie with the given value and CookieConfig
// properties (name, max-age, etc.).
//
// The MaxAge field is used to determine whether an Expires field should be
// added for Internet Explorer compatability and what its value should be.
func (cm *CookieMaker) NewCookie(value string) *http.Cookie {
	cookie := &http.Cookie{
		Value:    value,
		Name:     cm.Name,
		Domain:   cm.Domain,
		Path:     cm.Path,
		MaxAge:   cm.MaxAge,
		HttpOnly: cm.HTTPOnly,
		Secure:   cm.Secure,
	}
	// IE <9 does not understand MaxAge, set Expires if MaxAge is non-zero.
	// if expires, ok := expiresTime(config.MaxAge); ok {
	// 	cookie.Expires = expires
	// }
	return cookie
}
