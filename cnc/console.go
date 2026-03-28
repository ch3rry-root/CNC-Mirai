package main

import (
	"net"
	"time"
)

type Console interface {
	Read(b []byte) (int, error)
	Write(b []byte) (int, error)
	Close() error
	RemoteAddr() net.Addr
	SetReadDeadline(t time.Time) error
	SetWriteDeadline(t time.Time) error
	SetDeadline(t time.Time) error
}
