package application

import (
	"testing"

	"github.com/banzaicloud/pipeline/catalog"
	pkgCatalog "github.com/banzaicloud/pipeline/pkg/catalog"
)

func TestMergeRefValues(t *testing.T) {
	catalog := catalog.CatalogDetails{
		Spotguide: &pkgCatalog.SpotguideFile{
			Secrets: map[string]pkgCatalog.ApplicationSecret{
				"mysql-password": pkgCatalog.ApplicationSecret{
					Password: &pkgCatalog.ApplicationSecretPassword{},
				}},
		},
	}
	options := []pkgCatalog.ApplicationOptions{
		{
			Value: "s3cr3tPassword",
			Ref:   "#/secrets/mysql-password/password/password",
		},
	}

	err := mergeRefValues(&catalog, options)
	if err != nil {
		t.Fatal(err)
	}

	if catalog.Spotguide.Secrets["mysql-password"].Password.Password != options[0].Value {
		t.Fatal("catalog.Spotguide.Secrets[\"mysql-password\"].Password should have been set to", options[0].Value)
	}
}
