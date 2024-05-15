package model

type ISession interface {
	GetSubscriber() ISubscriber
	GetChannels() *map[IChannel]bool
	GetMessage() chan []byte
}
