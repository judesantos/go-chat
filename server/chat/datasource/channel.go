package datasource

import (
	"database/sql"
	"yt/chatbot/server/chat/model"
)

type Channel struct {
	model.IChannel
	Id      string
	Name    string
	Private bool
}

func (m *Channel) GetId() string {
	return m.Id
}

func (m *Channel) GetName() string {
	return m.Name
}

func (m *Channel) IsPrivate() bool {
	return m.Private
}

type ChannelSqlite struct {
	model.IChannelDS
	DbConn *sql.DB
}

func (m *ChannelSqlite) Add(channel model.IChannel) error {

	sql := "INSERT INTO channel(id, name, private) VALUES(?, ?, ?)"
	var err error

	stmt, err := m.DbConn.Prepare(sql)
	if err != nil {
		return err
	}
	defer func() {
		stmt.Close()
	}()

	_, err = stmt.Exec(channel.GetId(), channel.GetName(), channel.IsPrivate())

	return err
}

func (m *ChannelSqlite) Get(chName string) (model.IChannel, error) {

	sqlStmt := "SELECT id, name, private FROM channel WHERE name = ? LIMIT 1"

	channel := &Channel{}
	row := m.DbConn.QueryRow(sqlStmt, chName)

	err := row.Scan(&channel.Id, &channel.Name, &channel.Private)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return channel, nil
}
