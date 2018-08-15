package database

import "github.com/pkg/errors"

// Config holds information necessary for connecting to a database.
type Config struct {
	Host string
	Port int
	User string
	Pass string
	Name string

	Role string

	Params map[string]string

	EnableLog bool
}

// Validate checks that the configuration is valid.
func (c Config) Validate() error {
	if c.Host == "" {
		return errors.New("database host is required")
	}

	if c.Port == 0 {
		return errors.New("database port is required")
	}

	if c.Role == "" {
		if c.User == "" {
			return errors.New("database user is required if no secret role is provided")
		}
	}

	if c.Name == "" {
		return errors.New("database name is required")
	}

	return nil
}
