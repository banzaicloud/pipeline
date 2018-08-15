package database

import "testing"

func TestConfig_Validate(t *testing.T) {
	tests := map[string]Config{
		"database host is required": {
			Port: 3306,
			User: "root",
			Pass: "",
			Name: "database",
		},
		"database port is required": {
			Host: "localhost",
			User: "root",
			Pass: "",
			Name: "database",
		},
		"database user is required if no secret role is provided": {
			Host: "localhost",
			Port: 3306,
			Pass: "",
			Name: "database",
		},
		"database name is required": {
			Host: "localhost",
			Port: 3306,
			User: "root",
			Pass: "",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			err := test.Validate()

			if err.Error() != name {
				t.Errorf("expected error %q, got: %q", name, err.Error())
			}
		})
	}
}

func TestGetDSN(t *testing.T) {
	config := Config{
		Host: "host",
		Port: 3306,
		User: "root",
		Pass: "",
		Name: "database",
		Params: map[string]string{
			"parseTime": "true",
		},
	}

	dsn, err := GetDSN(config)
	if err != nil {
		t.Fatal("unexpected error:", err.Error())
	}

	expectedDsn := "root:@tcp(host:3306)/database?parseTime=true"
	if dsn != expectedDsn {
		t.Errorf("expected DSN to be %q, got: %q", expectedDsn, dsn)
	}
}
