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
	"testing"

	"emperror.dev/errors"
	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/require"
)

type fakeEntity struct {
	ID           uint `gorm:"primary_key"`
	ExampleField string
}

func (fakeEntity) TableName() string {
	return "fake_entity_table_name"
}

type fakeNoopTestObject struct{}

func (t *fakeNoopTestObject) Errorf(format string, arguments ...interface{}) {}

func (t *fakeNoopTestObject) FailNow() {}

type fakeOtherEntity struct {
	ID         uint `gorm:"primary_key"`
	OtherField string
}

func (fakeOtherEntity) TableName() string {
	return "fake_other_entity_table_name"
}

func TestFakeDatabaseCreateTablesFromEntities(t *testing.T) {
	type inputType struct {
		database            *FakeDatabase
		t                   TestObject
		tableEntityPointers []interface{}
	}

	testCases := []struct {
		caseDescription string
		input           inputType
	}{
		{
			caseDescription: "nil database -> nil database success",
			input: inputType{
				database: nil,
			},
		},
		{
			caseDescription: "nil t -> original database success",
			input: inputType{
				database: NewFakeDatabase(t),
				t:        nil,
			},
		},
		{
			caseDescription: "nil entities -> empty database success",
			input: inputType{
				database:            NewFakeDatabase(t),
				t:                   &fakeNoopTestObject{},
				tableEntityPointers: nil,
			},
		},
		{
			caseDescription: "empty entities -> empty database success",
			input: inputType{
				database:            NewFakeDatabase(t),
				t:                   &fakeNoopTestObject{},
				tableEntityPointers: []interface{}{},
			},
		},
		{
			caseDescription: "not empty entities -> not empty database success",
			input: inputType{
				database: NewFakeDatabase(t),
				t:        &fakeNoopTestObject{},
				tableEntityPointers: []interface{}{
					&fakeEntity{},
					&fakeOtherEntity{},
				},
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			for _, tableEntityPointer := range testCase.input.tableEntityPointers {
				require.Falsef(
					t,
					testCase.input.database.HasTable(tableEntityPointer),
					"expected table from entity %s is already present in input database",
					tableEntityPointer,
				)
			}

			actualDatabase := testCase.input.database.CreateTablesFromEntities(
				testCase.input.t,
				testCase.input.tableEntityPointers...,
			)

			if testCase.input.database == nil {
				require.Nil(t, actualDatabase)
			} else if testCase.input.t == nil {
				require.Equal(t, testCase.input.database, actualDatabase)
			} else {
				require.NotNil(t, actualDatabase)

				for _, tableEntityPointer := range testCase.input.tableEntityPointers {
					require.Truef(
						t,
						actualDatabase.HasTable(tableEntityPointer),
						"expected table from entity %s is missing from actual database",
						tableEntityPointer,
					)
				}
			}
		})
	}
}

func TestFakeDatabaseSaveEntities(t *testing.T) {
	type inputType struct {
		database       *FakeDatabase
		t              TestObject
		entityPointers []interface{}
	}

	testCases := []struct {
		caseDescription string
		input           inputType
	}{
		{
			caseDescription: "nil database -> nil database success",
			input: inputType{
				database: nil,
			},
		},
		{
			caseDescription: "nil t -> original database success",
			input: inputType{
				database: NewFakeDatabase(t),
				t:        nil,
			},
		},
		{
			caseDescription: "nil entities -> empty database success",
			input: inputType{
				database:       NewFakeDatabase(t),
				t:              &fakeNoopTestObject{},
				entityPointers: nil,
			},
		},
		{
			caseDescription: "empty entities -> empty database success",
			input: inputType{
				database:       NewFakeDatabase(t),
				t:              &fakeNoopTestObject{},
				entityPointers: []interface{}{},
			},
		},
		{
			caseDescription: "not empty entities -> not empty database success",
			input: inputType{
				database: NewFakeDatabase(t).CreateTablesFromEntities(t, &fakeEntity{}, &fakeOtherEntity{}),
				t:        &fakeNoopTestObject{},
				entityPointers: []interface{}{
					&fakeEntity{
						ID:           1,
						ExampleField: "example-field-1",
					},
					&fakeEntity{
						ID:           2,
						ExampleField: "example-field-2",
					},
					&fakeOtherEntity{
						ID:         1,
						OtherField: "other-field-1",
					},
					&fakeOtherEntity{
						ID:         2,
						OtherField: "other-field-2",
					},
				},
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			for _, inputEntityPointer := range testCase.input.entityPointers {
				err := (error)(nil)
				switch typedInputEntityPointer := inputEntityPointer.(type) {
				case *fakeEntity:
					err = testCase.input.database.Where(typedInputEntityPointer).First(&fakeEntity{}).Error
				case *fakeOtherEntity:
					err = testCase.input.database.Where(typedInputEntityPointer).First(&fakeOtherEntity{}).Error
				default:
					t.Errorf("unexpected input entity pointer type %+v", typedInputEntityPointer)
				}

				require.Error(t, err, "input entity %+v is already present in input database", inputEntityPointer)
				require.True(
					t,
					gorm.IsRecordNotFoundError(err),
					"unexpected database error encountered when checking input database: %+v",
					err,
				)
			}

			actualDatabase := testCase.input.database.SaveEntities(
				testCase.input.t,
				testCase.input.entityPointers...,
			)

			if testCase.input.database == nil {
				require.Nil(t, actualDatabase)
			} else if testCase.input.t == nil {
				require.Equal(t, testCase.input.database, actualDatabase)
			} else {
				require.NotNil(t, actualDatabase)

				for _, inputEntityPointer := range testCase.input.entityPointers {
					err := (error)(nil)
					switch typedInputEntityPointer := inputEntityPointer.(type) {
					case *fakeEntity:
						err = actualDatabase.Where(typedInputEntityPointer).First(&fakeEntity{}).Error
					case *fakeOtherEntity:
						err = actualDatabase.Where(typedInputEntityPointer).First(&fakeOtherEntity{}).Error
					default:
						t.Errorf("unexpected input entity pointer type %+v", typedInputEntityPointer)
					}

					require.NoError(t, err, "input entity %+v is missing from actual database", inputEntityPointer)
				}
			}
		})
	}
}

func TestFakeDatabaseSetError(t *testing.T) {
	type inputType struct {
		database *FakeDatabase
		t        TestObject
		err      error
	}

	testCases := []struct {
		caseDescription string
		input           inputType
	}{
		{
			caseDescription: "nil database -> nil database success",
			input: inputType{
				database: nil,
			},
		},
		{
			caseDescription: "nil t -> original database success",
			input: inputType{
				database: NewFakeDatabase(t),
				t:        nil,
			},
		},
		{
			caseDescription: "nil error -> database no error success",
			input: inputType{
				database: NewFakeDatabase(t),
				t:        &fakeNoopTestObject{},
				err:      nil,
			},
		},
		{
			caseDescription: "not nil error -> database error success",
			input: inputType{
				database: NewFakeDatabase(t),
				t:        &fakeNoopTestObject{},
				err:      errors.New("test error"),
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			if testCase.input.database != nil &&
				testCase.input.t != nil {
				require.NoError(t, testCase.input.database.DB.Error)
			}

			actualDatabase := testCase.input.database.SetError(
				testCase.input.t,
				testCase.input.err,
			)

			if testCase.input.database == nil {
				require.Nil(t, actualDatabase)
			} else if testCase.input.t == nil {
				require.Equal(t, testCase.input.database, actualDatabase)
			} else {
				require.NotNil(t, actualDatabase)
				require.NotNil(t, actualDatabase.DB)

				if testCase.input.err == nil {
					require.NoError(t, actualDatabase.DB.Error)
				} else {
					require.EqualError(t, actualDatabase.DB.Error, testCase.input.err.Error())
				}
			}
		})
	}
}

func TestNewFakeDatabase(t *testing.T) {
	testCases := []struct {
		caseDescription        string
		isExpectingNilDatabase bool
		t                      TestObject
	}{
		{
			caseDescription:        "nil t -> nil database success",
			isExpectingNilDatabase: true,
			t:                      nil,
		},
		{
			caseDescription:        "not nil t -> not nil database success",
			isExpectingNilDatabase: false,
			t:                      &fakeNoopTestObject{},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			actualFakeDatabase := NewFakeDatabase(testCase.t)

			if testCase.isExpectingNilDatabase {
				require.Nil(t, actualFakeDatabase)
			} else {
				require.NotNil(t, actualFakeDatabase)
				require.NotNil(t, actualFakeDatabase.DB)
			}
		})
	}
}
