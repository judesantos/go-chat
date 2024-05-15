package model

type ISubscriber interface {
	GetId() string
	GetName() string
}

type ISubscriberDS interface {
	Add(subscriber ISubscriber) error
	Remove(subscriberId string) error
	Get(subscriberId string) (ISubscriber, error)
	GetAll() ([]ISubscriber, error)
}
