package model

type IChannel interface {
	GetId() string
	GetName() string
	IsPrivate() bool
}

type IChannelDS interface {
	Add(channel IChannel) error
	Get(chName string) (IChannel, error)
	Remove(chName string) error
}
