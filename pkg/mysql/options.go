package mysql

import "time"

type Option func(*Mysql)

// WithMaxOpenConns 配置最大连接数。
func WithMaxOpenConns(size int) Option {
	return func(m *Mysql) {
		m.maxOpenConns = size
	}
}

// WithMaxIdleConns 配置最大空闲连接数。
func WithMaxIdleConns(size int) Option {
	return func(m *Mysql) {
		m.maxIdleConns = size
	}
}

// WithConnAttempts 配置连接重试次数。
func WithConnAttempts(attempts int) Option {
	return func(m *Mysql) {
		m.connAttempts = attempts
	}
}

// WithConnTimeout 配置连接重试间隔。
func WithConnTimeout(timeout time.Duration) Option {
	return func(m *Mysql) {
		m.connTimeout = timeout
	}
}
