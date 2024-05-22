package datasource

import (
	"database/sql"
	"yt/chat/server/chat/model"
)

type Subscriber struct {
	model.ISubscriber `json:"-"`
	Id                string `json:"id"`
	Name              string `json:"name"`
}

func (m *Subscriber) GetId() string {
	return m.Id
}

func (m *Subscriber) GetName() string {
	return m.Name
}

type SubscriberSqlite struct {
	model.ISubscriberDS
	DbConn *sql.DB
}

func (m *SubscriberSqlite) Add(subscriber model.ISubscriber) error {

	sqlStmt := "INSERT INTO subscriber(id, name) VALUES($1, $2)"

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

func (m *SubscriberSqlite) Remove(name string) error {

	sqlStmt := "DELETE FROM subscriber WHERE name = $1"

	stmt, err := m.DbConn.Prepare(sqlStmt)
	if err != nil {
		return err
	}
	defer func() {
		stmt.Close()
	}()

	_, err = stmt.Exec(name)

	return err
}

func (m *SubscriberSqlite) Get(name string) (model.ISubscriber, error) {

	sqlStmt := "SELECT id, name FROM subscriber where name = $1 LIMIT 1"

	row := m.DbConn.QueryRow(sqlStmt, name)

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

func (m *SubscriberSqlite) GetAll() ([]model.ISubscriber, error) {

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
