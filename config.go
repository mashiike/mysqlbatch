package mysqlbatch

import (
	"fmt"
	"strings"
)

// Config is a connection setting to MySQL.
// Exists to generate a Golang connection DSN to MySQL
type Config struct {
	DSN      string
	User     string
	Password string
	Host     string
	Port     int
	Database string
}

// NewDefaultConfig returns the config for connecting to the local MySQL server
func NewDefaultConfig() *Config {
	return &Config{
		User: "root",
		Host: "127.0.0.1",
		Port: 3306,
	}
}

// GetDSN returns a DSN dedicated to connecting to MySQL.
func (c *Config) GetDSN() string {
	if c.DSN == "" {
		return fmt.Sprintf(
			"%s:%s@tcp(%s:%d)/%s?parseTime=true",
			c.User,
			c.Password,
			c.Host,
			c.Port,
			c.Database,
		)
	}
	return strings.TrimPrefix(c.DSN, "mysql://")
}
