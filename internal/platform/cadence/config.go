package cadence

import (
	"fmt"

	"github.com/pkg/errors"
)

// Config holds information necessary for connecting to Cadence.
type Config struct {
	// Cadence connection details
	Host string
	Port int

	Domain string

	// Client identity (not used for now; falls back to machine host name)
	Identity string
}

// Validate checks that the configuration is valid.
func (c Config) Validate() error {
	if c.Host == "" {
		return errors.New("cadence host is required")
	}

	if c.Port == 0 {
		return errors.New("cadence port is required")
	}

	if c.Domain == "" {
		return errors.New("cadence domain is required")
	}

	return nil
}

// Addr returns the Cadence connection address.
func (c Config) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}
