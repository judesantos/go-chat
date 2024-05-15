package datasource

import (
	"database/sql"
	"yt/chatbot/server/chat/model"
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

	sqlStmt := "INSERT INTO subscriber(id, name) VALUES(?,?)"

	stmt, err := m.DbConn.Prepare(sqlStmt)
	if err != nil {
		return err
	}

	_, err = stmt.Exec(subscriber.GetId(), subscriber.GetName())

	return err
}

func (m *SubscriberSqlite) Remove(subscriberId string) error {

	sqlStmt := "DELETE FROM subscriber WHERE id = ?"

	stmt, err := m.DbConn.Prepare(sqlStmt)
	if err != nil {
		return err
	}

	_, err = stmt.Exec(subscriberId)

	return err
}

func (m *SubscriberSqlite) Get(subscriberId string) (model.ISubscriber, error) {

	sqlStmt := "SELECT id, name FROM subscriber where id = ? LIMIT 1"

	row := m.DbConn.QueryRow(sqlStmt, subscriberId)

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