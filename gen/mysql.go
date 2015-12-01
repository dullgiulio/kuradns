package gen

import (
	"database/sql"
	"errors"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"

	"github.com/dullgiulio/kuradns/cfg"
)

type mysql struct {
	ch   chan RawEntry
	rows *sql.Rows
}

func newMysql(c *cfg.Config) (*mysql, error) {
	usr, ok := c.Get("config.user")
	if !ok {
		return nil, errors.New("mysql user not specified")
	}
	pwd, ok := c.Get("config.password")
	if !ok {
		return nil, errors.New("mysql password not specified")
	}
	dbname, ok := c.Get("config.database")
	if !ok {
		return nil, errors.New("mysql database name not specified")
	}
	query, ok := c.Get("config.query")
	if !ok {
		return nil, errors.New("mysql query not specified")
	}
	host := c.GetVal("config.host", "localhost")
	port := c.GetVal("config.port", "3306")
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", usr, pwd, host, port, dbname)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to mysql[%s]: %s", dsn, err)
	}
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query on mysql[%s]: %s", dsn, err)
	}
	m := &mysql{
		ch:   make(chan RawEntry, 100),
		rows: rows,
	}
	// rows is closed in run()
	go m.run()
	return m, nil
}

func (m *mysql) run() {
	entry := MakeRawEntry("", "")
	defer m.rows.Close()
	for m.rows.Next() {
		if err := m.rows.Scan(&entry.Source, &entry.Target); err != nil {
			log.Printf("[dns] mysql: error reading rows: %s", err)
			continue
		}
		m.ch <- entry
	}
	close(m.ch)
}

func (m *mysql) Generate() RawEntry {
	return <-m.ch
}
