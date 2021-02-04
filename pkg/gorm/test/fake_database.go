// Copyright © 2020 Banzai Cloud
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package test

import (
	"github.com/jinzhu/gorm"

	// SQLite driver used for test fake purposes.
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/stretchr/testify/require"
)

// FakeDatabase provides a minimal in-memory database implementation to use during
// tests in place of a GORM database.
type FakeDatabase struct {
	*gorm.DB
}

// NewFakeDatabase instantiates a fake database.
func NewFakeDatabase(t TestObject) *FakeDatabase {
	if t == nil {
		return nil
	}

	gormDatabase, err := gorm.Open("sqlite3", "file::memory:")
	require.NoError(t, err)

	return &FakeDatabase{
		DB: gormDatabase,
	}
}

// CreateTableFromEntity creates new tables in the database using the provided
// entity pointers.
func (database *FakeDatabase) CreateTablesFromEntities(t TestObject, tableEntityPointers ...interface{}) *FakeDatabase {
	if database == nil ||
		t == nil {
		return database
	}

	for _, tableEntityPointer := range tableEntityPointers {
		if !database.HasTable(tableEntityPointer) {
			require.NoError(t, database.AutoMigrate(tableEntityPointer).Error)
		}
	}

	return database
}

// SaveEntities takes a collection of entity pointers and saves them to the fake
// database.
func (database *FakeDatabase) SaveEntities(t TestObject, entityPointers ...interface{}) *FakeDatabase {
	if database == nil ||
		t == nil {
		return database
	}

	for _, entity := range entityPointers {
		require.NoError(t, database.Save(entity).Error)
	}

	return database
}

func (database *FakeDatabase) SetError(t TestObject, err error) *FakeDatabase {
	if database == nil ||
		t == nil {
		return database
	}

	database.DB.Error = err

	return database
}
