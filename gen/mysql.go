// Copyright 2016 Giulio Iotti. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

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
	ch    chan *RawEntry
	errch chan error
	rows  *sql.Rows
}

func newMysql(c *cfg.Config) (*mysql, error) {
	usr, ok := c.Get("config.user")
	if !ok || usr == "" {
		return nil, errors.New("mysql user not specified")
	}
	pwd, ok := c.Get("config.password")
	if !ok || pwd == "" {
		return nil, errors.New("mysql password not specified")
	}
	dbname, ok := c.Get("config.database")
	if !ok || dbname == "" {
		return nil, errors.New("mysql database name not specified")
	}
	query, ok := c.Get("config.query")
	if !ok || query == "" {
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
		ch:    make(chan *RawEntry, 100),
		errch: make(chan error),
		rows:  rows,
	}
	// rows is closed in run()
	go func() {
		m.run()
		db.Close()
	}()
	return m, nil
}

func (m *mysql) run() {
	defer m.rows.Close()
	for m.rows.Next() {
		entry := NewRawEntry("", "")
		if err := m.rows.Scan(&entry.Source, &entry.Target); err != nil {
			m.errch <- fmt.Errorf("mysql: error reading rows: %s", err)
			continue
		}
		if entry.Source == "" && entry.Target == "" {
			log.Printf("[dns] mysql: skipping empty entry from database")
			continue
		}
		m.ch <- entry
	}
	close(m.ch)
}

func (m *mysql) Generate() (*RawEntry, error) {
	select {
	case re := <-m.ch:
		return re, nil
	case err := <-m.errch:
		return nil, err
	}
}
