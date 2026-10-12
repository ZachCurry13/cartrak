package server

import (
	"net/http"
	"net/http/cookiejar"
)

func cookieJar() (http.CookieJar, error) { return cookiejar.New(nil) }
