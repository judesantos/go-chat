package datasource

import (
	"database/sql"
	"yt/chat/server/chat/model"
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

type ChannelPgsql struct {
	model.IChannelDS
	DbConn *sql.DB
}

func (m *ChannelPgsql) Add(channel model.IChannel) error {

	sql := "INSERT INTO channel(id, name, private) VALUES($1, $2, $3)"
	var err error

	stmt, err := m.DbConn.Prepare(sql)
	if err != nil {
		return err
	}
	defer func() {
		stmt.Close()
	}()

	private := 0
	if channel.IsPrivate() {
		private = 1
	}

	_, err = stmt.Exec(channel.GetId(), channel.GetName(), private)

	return err
}

func (m *ChannelPgsql) Get(chName string) (model.IChannel, error) {

	sqlStmt := "SELECT id, name, private FROM channel WHERE name = $1 LIMIT 1"

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
