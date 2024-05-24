package datasource

import (
	"database/sql"
	"yt/chat/server/chat/model"
)

type SubscriberType string

const (
	SUBSCRIBER_TYPE_ANONYMOUS = "anonymous"
	SUBSCRIBER_TYPE_LOGIN     = "login"
)

type Subscriber struct {
	model.ISubscriber `json:"-"`
	Id                string `json:"id"`
	Name              string `json:"name"`
	Password          string `json:"password"`
	Email             string `json:"email"`
	Type              string `json:"-"`
}

func (m *Subscriber) GetId() string {
	return m.Id
}

func (m *Subscriber) GetName() string {
	return m.Name
}

func (m *Subscriber) GetPassword() string {
	return m.Password
}

func (m *Subscriber) GetEmail() string {
	return m.Email
}

type SubscriberPgsql struct {
	model.ISubscriberDS
	DbConn *sql.DB
}

func (m *SubscriberPgsql) Add(subscriber model.ISubscriber) error {

	var sqlStmt string
	if subscriber.(*Subscriber).Type == SUBSCRIBER_TYPE_ANONYMOUS {
		sqlStmt = "INSERT INTO transient(name, email) VALUES($1, $2)"
	} else {
		sqlStmt = "INSERT INTO subscriber(name, password, email) VALUES($1, $2, $3)"
	}

	stmt, err := m.DbConn.Prepare(sqlStmt)
	if err != nil {
		return err
	}
	defer func() {
		stmt.Close()
	}()

	if subscriber.(*Subscriber).Type == SUBSCRIBER_TYPE_ANONYMOUS {
		_, err = stmt.Exec(subscriber.GetName())
	} else {
		_, err = stmt.Exec(subscriber.GetName(), subscriber.GetPassword(), subscriber.GetEmail())
	}

	return err
}

func (m *SubscriberPgsql) Remove(subscriber model.ISubscriber) error {

	var sqlStmt string
	if subscriber.(*Subscriber).Type == SUBSCRIBER_TYPE_ANONYMOUS {
		sqlStmt = "DELETE FROM transientr WHERE name = $1"
	} else {
		sqlStmt = "DELETE FROM subscriber WHERE name = $1"
	}

	stmt, err := m.DbConn.Prepare(sqlStmt)
	if err != nil {
		return err
	}
	defer func() {
		stmt.Close()
	}()

	_, err = stmt.Exec(subscriber.GetName())

	return err
}

// Works only for subscribers - registered users
func (m *SubscriberPgsql) GetLoginInfo(subscriber model.ISubscriber) (model.ISubscriber, error) {

	sqlStmt := "SELECT name, password FROM subscriber where name = $1 LIMIT 1"

	row := m.DbConn.QueryRow(sqlStmt, subscriber.GetName())

	var subs Subscriber

	err := row.Scan(&subs.Name, &subs.Password)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &subs, nil
}

func (m *SubscriberPgsql) Get(subscriber model.ISubscriber) (model.ISubscriber, error) {

	var sqlStmt string
	if subscriber.(*Subscriber).Type == SUBSCRIBER_TYPE_ANONYMOUS {
		sqlStmt = "SELECT id, name FROM transient where name = $1 LIMIT 1"
	} else {
		sqlStmt = "SELECT id, name, email FROM subscriber where name = $1 LIMIT 1"
	}

	row := m.DbConn.QueryRow(sqlStmt, subscriber.GetName())

	var subs Subscriber

	err := row.Scan(&subs.Id, &subs.Name, &subs.Email)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &subs, nil
}
