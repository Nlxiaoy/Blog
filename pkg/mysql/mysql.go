package mysql

import (
	"fmt"
	"log"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

const (
	_defaultMaxIdleConns = 10
	_defaultMaxOpenConns = 100
	_defaultConnAttempts = 10
	_defaultConnTimeout  = time.Second
)

type Mysql struct {
	maxIdleConns int
	maxOpenConns int
	connAttempts int
	connTimeout  time.Duration

	DB *gorm.DB
}

func (m *Mysql) Close() {
	if m.DB != nil {
		sqlDB, _ := m.DB.DB()
		_ = sqlDB.Close()
		fmt.Println("Mysql: connection closed")
	}
}

func New(host string, port int, user, password, dbname, config string, opts ...Option) (*Mysql, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?%s",
		user,
		password,
		host,
		port,
		dbname,
		config,
	)

	m := &Mysql{
		maxIdleConns: _defaultMaxIdleConns,
		maxOpenConns: _defaultMaxOpenConns,
		connAttempts: _defaultConnAttempts,
		connTimeout:  _defaultConnTimeout,
	}

	for _, opt := range opts {
		opt(m)
	}

	var err error
	for m.connAttempts > 0 {
		m.DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
		if err == nil {
			sqlDB, _ := m.DB.DB()
			sqlDB.SetMaxOpenConns(m.maxOpenConns)
			sqlDB.SetMaxIdleConns(m.maxIdleConns)
			break
		}

		log.Printf("mysql: connect retry, attempts left: %d", m.connAttempts)
		time.Sleep(m.connTimeout)
		m.connAttempts--
	}

	if err != nil {
		return nil, fmt.Errorf("Mysql - New - connAttempts == 0: %w", err)
	}

	return m, nil
}
