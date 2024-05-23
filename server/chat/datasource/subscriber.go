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

type SubscriberPgsql struct {
	model.ISubscriberDS
	DbConn *sql.DB
}

func (m *SubscriberPgsql) Add(subscriber model.ISubscriber) error {

	var sqlStmt string
	if subscriber.(*Subscriber).Type == SUBSCRIBER_TYPE_ANONYMOUS {
		sqlStmt = "INSERT INTO transient(id, name) VALUES($1, $2)"
	} else {
		sqlStmt = "INSERT INTO subscriber(id, name) VALUES($1, $2)"
	}

	stmt, err := m.DbConn.Prepare(sqlStmt)
	if err != nil {
		return err
	}
	defer func() {
		stmt.Close()
	}()

	_, err = stmt.Exec(subscriber.GetId(), subscriber.GetName())

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

func (m *SubscriberPgsql) Get(subscriber model.ISubscriber) (model.ISubscriber, error) {

	var sqlStmt string
	if subscriber.(*Subscriber).Type == SUBSCRIBER_TYPE_ANONYMOUS {
		sqlStmt = "SELECT id, name FROM transient where name = $1 LIMIT 1"
	} else {
		sqlStmt = "SELECT id, name FROM subscriber where name = $1 LIMIT 1"
	}

	row := m.DbConn.QueryRow(sqlStmt, subscriber.GetName())

	var subs Subscriber

	err := row.Scan(&subs.Id, &subs.Name)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &subs, nil
}

func (m *SubscriberPgsql) GetAll() ([]model.ISubscriber, error) {

	sqlStmt := "SELECT id, name FROM subscriber"

	rows, err := m.DbConn.Query(sqlStmt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []model.ISubscriber

	for rows.Next() {
		sub := &Subscriber{}
		rows.Scan(&sub.Id, &sub.Name)
		subs = append(subs, sub)
	}

	return subs, nil
}
