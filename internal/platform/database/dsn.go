package database

import (
	"fmt"

	"github.com/banzaicloud/bank-vaults/database"
)

// GetDSN returns a DSN string from a config.
func GetDSN(c Config) (string, error) {
	dsn := fmt.Sprintf("@tcp(%s:%d)/%s", c.Host, c.Port, c.Name)

	if c.Role != "" {
		var err error

		dsn, err = database.DynamicSecretDataSource("mysql", c.Role+dsn)
		if err != nil {
			return "", err
		}
	} else {
		dsn = fmt.Sprintf("%s:%s%s", c.User, c.Pass, dsn)
	}

	var params string

	if len(c.Params) > 0 {
		var query string

		for key, value := range c.Params {
			if query != "" {
				query += "&"
			}

			query += key + "=" + value
		}

		params = "?" + query
	}

	dsn = dsn + params

	return dsn, nil
}
