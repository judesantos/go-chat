package model

type ISubscriber interface {
	GetId() string
	GetName() string
	GetPassword() string
	GetEmail() string
}

type ISubscriberDS interface {
	Add(subscriber ISubscriber) error
	Remove(subscriber ISubscriber) error
	Get(subscriber ISubscriber) (ISubscriber, error)
	GetAll() ([]ISubscriber, error)
}
