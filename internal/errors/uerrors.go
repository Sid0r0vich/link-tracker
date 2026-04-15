package uerrors

import "errors"

var ErrChatAlreadyExists = errors.New("chat already exists")
var ErrLinkAlreadyExists = errors.New("link already exists")
var ErrChatNotExists = errors.New("chat not exists")
var ErrLinkNotFound = errors.New("link not found")
var ErrChatNotExistsOrLinkNotFound = errors.New("chat not exists or link not found")
var ErrBadURL = errors.New("bad URL")
var ErrTooManyRequests = errors.New("too many requests")
var ErrBadToken = errors.New("bad token")
var ErrInternal = errors.New("internal server error")
var ErrAPIUnavailable = errors.New("API is unavailable")
var ErrAPINotAlowed = errors.New("API is not allowed")
