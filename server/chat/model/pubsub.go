package model

type IPubSub interface {
	Publish(channelName string, message string) error
	Subscribe(channel IChannel) (IPubSubChannel, error)
}

type IPubSubChannel interface {
	BroadCastMessage()
}
