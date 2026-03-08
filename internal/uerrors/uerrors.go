package uerrors

import "errors"

var ErrChatAlreadyExists = errors.New("chat already exists")
var ErrLinkAlreadyExists = errors.New("link already exists")
var ErrChatNotExists = errors.New("chat not exists")
var ErrLinkNotFound = errors.New("link not found")
var ErrChatNotExistsOrLinkNotFound = errors.New("chat not exists or link not found")
