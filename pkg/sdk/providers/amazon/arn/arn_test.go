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

package arn

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAccountID(t *testing.T) {
	testCases := []struct {
		arn               string
		caseName          string
		expectedAccountID string
	}{
		{
			arn:               "",
			caseName:          "empty string",
			expectedAccountID: "",
		},
		{
			arn:               "not-arn",
			caseName:          "invalid prefix",
			expectedAccountID: "",
		},
		{
			arn:               "arn:not:enough:sections",
			caseName:          "invalid section structure, not enough sections",
			expectedAccountID: "",
		},
		{
			arn:               "arn:valid:example:with:6:sections",
			caseName:          "valid arn, structural example",
			expectedAccountID: "6",
		},
		{
			arn:               "arn:aws:elasticbeanstalk:us-east-1:123456789012:environment/My App/MyEnvironment",
			caseName:          "valid arn, real example space in path",
			expectedAccountID: "123456789012",
		},
		{
			arn:               "arn:aws:iam::123456789012:user/David",
			caseName:          "valid arn, real example simple type and name",
			expectedAccountID: "123456789012",
		},
		{
			arn:               "arn:aws:rds:eu-west-1:123456789012:db:mysql-db",
			caseName:          "valid arn, real example section separator path / more than usual section count",
			expectedAccountID: "123456789012",
		},
		{
			arn:               "arn:aws:s3:::my_corporate_bucket/exampleobject.png",
			caseName:          "valid arn, real example no type",
			expectedAccountID: "",
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseName, func(t *testing.T) {
			actualAccountID := AccountID(testCase.arn)

			require.Equal(t, testCase.expectedAccountID, actualAccountID)
		})
	}
}

func TestIsARN(t *testing.T) {
	testCases := []struct {
		caseName      string
		candidate     string
		expectedIsARN bool
	}{
		{
			caseName:      "empty string",
			candidate:     "",
			expectedIsARN: false,
		},
		{
			caseName:      "invalid prefix",
			candidate:     "not-arn",
			expectedIsARN: false,
		},
		{
			caseName:      "invalid section structure, not enough sections",
			candidate:     "arn:not:enough:sections",
			expectedIsARN: false,
		},
		{
			caseName:      "valid arn, structural example",
			candidate:     "arn:valid:example:with:6:sections",
			expectedIsARN: true,
		},
		{
			caseName:      "valid arn, real example space in path",
			candidate:     "arn:aws:elasticbeanstalk:us-east-1:123456789012:environment/My App/MyEnvironment",
			expectedIsARN: true,
		},
		{
			caseName:      "valid arn, real example simple type and name",
			candidate:     "arn:aws:iam::123456789012:user/David",
			expectedIsARN: true,
		},
		{
			caseName:      "valid arn, real example section separator path / more than usual section count",
			candidate:     "arn:aws:rds:eu-west-1:123456789012:db:mysql-db",
			expectedIsARN: true,
		},
		{
			caseName:      "valid arn, real example no type",
			candidate:     "arn:aws:s3:::my_corporate_bucket/exampleobject.png",
			expectedIsARN: true,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseName, func(t *testing.T) {
			actualIsARN := IsARN(testCase.candidate)

			require.Equal(t, testCase.expectedIsARN, actualIsARN)
		})
	}
}

func TestNewARN(t *testing.T) {
	type inputType struct {
		accountID            string
		partition            string
		region               string
		resourceName         string
		resourcePathOrParent string
		resourceQualifier    string
		resourceType         string
		service              string
	}

	testCases := []struct {
		caseName    string
		expectedARN string
		input       inputType
	}{
		{
			caseName:    "empty values success",
			expectedARN: "arn:::::",
			input: inputType{
				accountID:            "",
				partition:            "",
				region:               "",
				resourceName:         "",
				resourcePathOrParent: "",
				resourceQualifier:    "",
				resourceType:         "",
				service:              "",
			},
		},
		{
			caseName:    "only name success",
			expectedARN: "arn:partition:service:region:accountID:resource-name",
			input: inputType{
				accountID:            "accountID",
				partition:            "partition",
				region:               "region",
				resourceName:         "resource-name",
				resourcePathOrParent: "",
				resourceQualifier:    "",
				resourceType:         "",
				service:              "service",
			},
		},
		{
			caseName:    "name and type success",
			expectedARN: "arn:partition:service:region:accountID:resource-typeresource-name",
			input: inputType{
				accountID:            "accountID",
				partition:            "partition",
				region:               "region",
				resourceName:         "resource-name",
				resourcePathOrParent: "",
				resourceQualifier:    "",
				resourceType:         "resource-type",
				service:              "service",
			},
		},
		{
			caseName:    "name and path success",
			expectedARN: "arn:partition:service:region:accountID:/resource/path/resource-name",
			input: inputType{
				accountID:            "accountID",
				partition:            "partition",
				region:               "region",
				resourceName:         "resource-name",
				resourcePathOrParent: "/resource/path/",
				resourceQualifier:    "",
				resourceType:         "",
				service:              "service",
			},
		},
		{
			caseName:    "name and qualifier success",
			expectedARN: "arn:partition:service:region:accountID:resource-name:resource-qualifier",
			input: inputType{
				accountID:            "accountID",
				partition:            "partition",
				region:               "region",
				resourceName:         "resource-name",
				resourcePathOrParent: "",
				resourceQualifier:    "resource-qualifier",
				resourceType:         "",
				service:              "service",
			},
		},
		{
			caseName:    "name, type and path success",
			expectedARN: "arn:partition:service:region:accountID:resource-type/resource/path/resource-name",
			input: inputType{
				accountID:            "accountID",
				partition:            "partition",
				region:               "region",
				resourceName:         "resource-name",
				resourcePathOrParent: "/resource/path/",
				resourceQualifier:    "",
				resourceType:         "resource-type",
				service:              "service",
			},
		},
		{
			caseName:    "name, type and qualifier success",
			expectedARN: "arn:partition:service:region:accountID:resource-typeresource-name:resource-qualifier",
			input: inputType{
				accountID:            "accountID",
				partition:            "partition",
				region:               "region",
				resourceName:         "resource-name",
				resourcePathOrParent: "",
				resourceQualifier:    "resource-qualifier",
				resourceType:         "resource-type",
				service:              "service",
			},
		},
		{
			caseName:    "name, path and qualifier success",
			expectedARN: "arn:partition:service:region:accountID:/resource/path/resource-name:resource-qualifier",
			input: inputType{
				accountID:            "accountID",
				partition:            "partition",
				region:               "region",
				resourceName:         "resource-name",
				resourcePathOrParent: "/resource/path/",
				resourceQualifier:    "resource-qualifier",
				resourceType:         "",
				service:              "service",
			},
		},
		{
			caseName:    "everything success",
			expectedARN: "arn:partition:service:region:accountID:resource-type/resource/path/resource-name:resource-qualifier",
			input: inputType{
				accountID:            "accountID",
				partition:            "partition",
				region:               "region",
				resourceName:         "resource-name",
				resourcePathOrParent: "/resource/path/",
				resourceQualifier:    "resource-qualifier",
				resourceType:         "resource-type",
				service:              "service",
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseName, func(t *testing.T) {
			actualARN := NewARN(
				testCase.input.partition,
				testCase.input.service,
				testCase.input.region,
				testCase.input.accountID,
				testCase.input.resourceType,
				testCase.input.resourcePathOrParent,
				testCase.input.resourceName,
				testCase.input.resourceQualifier,
			)

			require.Equal(t, testCase.expectedARN, actualARN)
		})
	}
}

func TestPartition(t *testing.T) {
	testCases := []struct {
		arn               string
		caseName          string
		expectedPartition string
	}{
		{
			arn:               "",
			caseName:          "empty string success",
			expectedPartition: "",
		},
		{
			arn:               "not-arn",
			caseName:          "invalid prefix success",
			expectedPartition: "",
		},
		{
			arn:               "arn:not:enough:sections",
			caseName:          "invalid section structure, not enough sections success",
			expectedPartition: "",
		},
		{
			arn:               "arn:valid:example:with:6:sections",
			caseName:          "valid arn, structural example success",
			expectedPartition: "valid",
		},
		{
			arn:               "arn:aws:elasticbeanstalk:us-east-1:123456789012:environment/My App/MyEnvironment",
			caseName:          "valid arn, real example space in path success",
			expectedPartition: "aws",
		},
		{
			arn:               "arn:aws:iam::123456789012:user/David",
			caseName:          "valid arn, real example simple type and name success",
			expectedPartition: "aws",
		},
		{
			arn:               "arn:aws:rds:eu-west-1:123456789012:db:mysql-db",
			caseName:          "valid arn, real example section separator path / more than usual section count success",
			expectedPartition: "aws",
		},
		{
			arn:               "arn:aws:s3:::my_corporate_bucket/exampleobject.png",
			caseName:          "valid arn, real example no type success",
			expectedPartition: "aws",
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseName, func(t *testing.T) {
			actualPartition := Partition(testCase.arn)

			require.Equal(t, testCase.expectedPartition, actualPartition)
		})
	}
}

func TestRegion(t *testing.T) {
	testCases := []struct {
		arn            string
		caseName       string
		expectedRegion string
	}{
		{
			arn:            "",
			caseName:       "empty string success",
			expectedRegion: "",
		},
		{
			arn:            "not-arn",
			caseName:       "invalid prefix success",
			expectedRegion: "",
		},
		{
			arn:            "arn:not:enough:sections",
			caseName:       "invalid section structure, not enough sections success",
			expectedRegion: "",
		},
		{
			arn:            "arn:valid:example:with:6:sections",
			caseName:       "valid arn, structural example success",
			expectedRegion: "with",
		},
		{
			arn:            "arn:aws:elasticbeanstalk:us-east-1:123456789012:environment/My App/MyEnvironment",
			caseName:       "valid arn, real example space in path success",
			expectedRegion: "us-east-1",
		},
		{
			arn:            "arn:aws:iam::123456789012:user/David",
			caseName:       "valid arn, real example simple type and name success",
			expectedRegion: "",
		},
		{
			arn:            "arn:aws:rds:eu-west-1:123456789012:db:mysql-db",
			caseName:       "valid arn, real example section separator path / more than usual section count success",
			expectedRegion: "eu-west-1",
		},
		{
			arn:            "arn:aws:s3:::my_corporate_bucket/exampleobject.png",
			caseName:       "valid arn, real example no type success",
			expectedRegion: "",
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseName, func(t *testing.T) {
			actualRegion := Region(testCase.arn)

			require.Equal(t, testCase.expectedRegion, actualRegion)
		})
	}
}

func TestResource(t *testing.T) {
	testCases := []struct {
		arn              string
		caseName         string
		expectedResource string
	}{
		{
			arn:              "",
			caseName:         "empty string success",
			expectedResource: "",
		},
		{
			arn:              "not-arn",
			caseName:         "invalid prefix success",
			expectedResource: "",
		},
		{
			arn:              "arn:not:enough:sections",
			caseName:         "invalid section structure, not enough sections success",
			expectedResource: "",
		},
		{
			arn:              "arn:valid:example:with:6:sections",
			caseName:         "valid arn, structural example success",
			expectedResource: "sections",
		},
		{
			arn:              "arn:aws:elasticbeanstalk:us-east-1:123456789012:environment/My App/MyEnvironment",
			caseName:         "valid arn, real example space in path success",
			expectedResource: "environment/My App/MyEnvironment",
		},
		{
			arn:              "arn:aws:iam::123456789012:user/David",
			caseName:         "valid arn, real example simple type and name success",
			expectedResource: "user/David",
		},
		{
			arn:              "arn:aws:rds:eu-west-1:123456789012:db:mysql-db",
			caseName:         "valid arn, real example section separator path / more than usual section count success",
			expectedResource: "db:mysql-db",
		},
		{
			arn:              "arn:aws:s3:::my_corporate_bucket/exampleobject.png",
			caseName:         "valid arn, real example no type success",
			expectedResource: "my_corporate_bucket/exampleobject.png",
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseName, func(t *testing.T) {
			actualResource := Resource(testCase.arn)

			require.Equal(t, testCase.expectedResource, actualResource)
		})
	}
}

func TestResourceName(t *testing.T) {
	testCases := []struct {
		arn                  string
		caseName             string
		expectedResourceName string
	}{
		{
			arn:                  "",
			caseName:             "empty string success",
			expectedResourceName: "",
		},
		{
			arn:                  "not-arn",
			caseName:             "invalid prefix success",
			expectedResourceName: "",
		},
		{
			arn:                  "arn:not:enough:sections",
			caseName:             "invalid section structure, not enough sections success",
			expectedResourceName: "",
		},
		{
			arn:                  "arn:valid:example:with:6:sections",
			caseName:             "valid arn, structural example success",
			expectedResourceName: "sections",
		},
		{
			arn:                  "arn:aws:elasticbeanstalk:us-east-1:123456789012:environment/My App/MyEnvironment",
			caseName:             "valid arn, real example space in path success",
			expectedResourceName: "MyEnvironment",
		},
		{
			arn:                  "arn:aws:iam::123456789012:user/David",
			caseName:             "valid arn, real example simple type and name success",
			expectedResourceName: "David",
		},
		{
			arn:                  "arn:aws:rds:eu-west-1:123456789012:db:mysql-db",
			caseName:             "valid arn, real example section separator path / more than usual section count success",
			expectedResourceName: "mysql-db",
		},
		{
			arn:                  "arn:aws:s3:::my_corporate_bucket/exampleobject.png",
			caseName:             "valid arn, real example no type success",
			expectedResourceName: "exampleobject.png",
		},
		{
			arn:                  "arn:valid:example:with:6:type/name",
			caseName:             "valid arn, real example / type separator success",
			expectedResourceName: "name",
		},
		{
			arn:                  "arn:valid:example:with:6:type:name",
			caseName:             "valid arn, real example : type separator success",
			expectedResourceName: "name",
		},
		{
			arn:                  "arn:valid:example:with:6:type/path/elements/name",
			caseName:             "valid arn, real example / type separator and path success",
			expectedResourceName: "name",
		},
		{
			arn:                  "arn:valid:example:with:6:type:path/elements/name",
			caseName:             "valid arn, real example : type separator and path success",
			expectedResourceName: "name",
		},
		{
			arn:                  "arn:valid:example:with:6:type/name:qualifier",
			caseName:             "valid arn, real example / type separator with qualifier success",
			expectedResourceName: "name",
		},
		{
			arn:                  "arn:valid:example:with:6:type:name:qualifier",
			caseName:             "valid arn, real example : type separator with qualifier success",
			expectedResourceName: "name",
		},
		{
			arn:                  "arn:valid:example:with:6:type/path/elements/name:qualifier",
			caseName:             "valid arn, real example / type separator, path and qualifier success",
			expectedResourceName: "name",
		},
		{
			arn:                  "arn:valid:example:with:6:type:path/elements/name:qualifier",
			caseName:             "valid arn, real example : type separator, path and qualifier success",
			expectedResourceName: "name",
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseName, func(t *testing.T) {
			actualResourceName := ResourceName(testCase.arn)

			require.Equal(t, testCase.expectedResourceName, actualResourceName)
		})
	}
}

func TestResourcePathOrParent(t *testing.T) {
	testCases := []struct {
		arn                          string
		caseName                     string
		expectedResourcePathOrParent string
	}{
		{
			arn:                          "",
			caseName:                     "empty string success",
			expectedResourcePathOrParent: "",
		},
		{
			arn:                          "not-arn",
			caseName:                     "invalid prefix success",
			expectedResourcePathOrParent: "",
		},
		{
			arn:                          "arn:not:enough:sections",
			caseName:                     "invalid section structure, not enough sections success",
			expectedResourcePathOrParent: "",
		},
		{
			arn:                          "arn:valid:example:with:6:sections",
			caseName:                     "valid arn, structural example success",
			expectedResourcePathOrParent: "",
		},
		{
			arn:                          "arn:aws:elasticbeanstalk:us-east-1:123456789012:environment/My App/MyEnvironment",
			caseName:                     "valid arn, real example space in path success",
			expectedResourcePathOrParent: "/My App/",
		},
		{
			arn:                          "arn:aws:iam::123456789012:user/David",
			caseName:                     "valid arn, real example simple type and name success",
			expectedResourcePathOrParent: "/",
		},
		{
			arn:                          "arn:aws:rds:eu-west-1:123456789012:db:mysql-db",
			caseName:                     "valid arn, real example section separator path / more than usual section count success",
			expectedResourcePathOrParent: ":",
		},
		{
			arn:                          "arn:aws:s3:::my_corporate_bucket/exampleobject.png",
			caseName:                     "valid arn, real example no type success",
			expectedResourcePathOrParent: "/",
		},
		{
			arn:                          "arn:valid:example:with:6:type/name",
			caseName:                     "valid arn, real example / type separator success",
			expectedResourcePathOrParent: "/",
		},
		{
			arn:                          "arn:valid:example:with:6:type:name",
			caseName:                     "valid arn, real example : type separator success",
			expectedResourcePathOrParent: ":",
		},
		{
			arn:                          "arn:valid:example:with:6:type/path/elements/name",
			caseName:                     "valid arn, real example / type separator and path success",
			expectedResourcePathOrParent: "/path/elements/",
		},
		{
			arn:                          "arn:valid:example:with:6:type:path/elements/name",
			caseName:                     "valid arn, real example : type separator and path success",
			expectedResourcePathOrParent: ":path/elements/",
		},
		{
			arn:                          "arn:valid:example:with:6:type/name:qualifier",
			caseName:                     "valid arn, real example / type separator with qualifier success",
			expectedResourcePathOrParent: "/",
		},
		{
			arn:                          "arn:valid:example:with:6:type:name:qualifier",
			caseName:                     "valid arn, real example : type separator with qualifier success",
			expectedResourcePathOrParent: ":",
		},
		{
			arn:                          "arn:valid:example:with:6:type/path/elements/name:qualifier",
			caseName:                     "valid arn, real example / type separator, path and qualifier success",
			expectedResourcePathOrParent: "/path/elements/",
		},
		{
			arn:                          "arn:valid:example:with:6:type:path/elements/name:qualifier",
			caseName:                     "valid arn, real example : type separator, path and qualifier success",
			expectedResourcePathOrParent: ":path/elements/",
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseName, func(t *testing.T) {
			actualResourcePathOrParent := ResourcePathOrParent(testCase.arn)

			require.Equal(t, testCase.expectedResourcePathOrParent, actualResourcePathOrParent)
		})
	}
}

func TestResourceQualifier(t *testing.T) {
	testCases := []struct {
		arn                       string
		caseName                  string
		expectedResourceQualifier string
	}{
		{
			arn:                       "",
			caseName:                  "empty string success",
			expectedResourceQualifier: "",
		},
		{
			arn:                       "not-arn",
			caseName:                  "invalid prefix success",
			expectedResourceQualifier: "",
		},
		{
			arn:                       "arn:not:enough:sections",
			caseName:                  "invalid section structure, not enough sections success",
			expectedResourceQualifier: "",
		},
		{
			arn:                       "arn:valid:example:with:6:sections",
			caseName:                  "valid arn, structural example success",
			expectedResourceQualifier: "",
		},
		{
			arn:                       "arn:aws:elasticbeanstalk:us-east-1:123456789012:environment/My App/MyEnvironment",
			caseName:                  "valid arn, real example space in path success",
			expectedResourceQualifier: "",
		},
		{
			arn:                       "arn:aws:iam::123456789012:user/David",
			caseName:                  "valid arn, real example simple type and name success",
			expectedResourceQualifier: "",
		},
		{
			arn:                       "arn:aws:rds:eu-west-1:123456789012:db:mysql-db",
			caseName:                  "valid arn, real example section separator path / more than usual section count success",
			expectedResourceQualifier: "",
		},
		{
			arn:                       "arn:aws:s3:::my_corporate_bucket/exampleobject.png",
			caseName:                  "valid arn, real example no type success",
			expectedResourceQualifier: "",
		},
		{
			arn:                       "arn:valid:example:with:6:type/name:qualifier",
			caseName:                  "valid arn, real example / type separator with qualifier success",
			expectedResourceQualifier: "qualifier",
		},
		{
			arn:                       "arn:valid:example:with:6:type:name:qualifier",
			caseName:                  "valid arn, real example : type separator with qualifier success",
			expectedResourceQualifier: "qualifier",
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseName, func(t *testing.T) {
			actualResourceQualifier := ResourceQualifier(testCase.arn)

			require.Equal(t, testCase.expectedResourceQualifier, actualResourceQualifier)
		})
	}
}

func TestResourceType(t *testing.T) {
	testCases := []struct {
		arn                  string
		caseName             string
		expectedResourceType string
	}{
		{
			arn:                  "",
			caseName:             "empty string success",
			expectedResourceType: "",
		},
		{
			arn:                  "not-arn",
			caseName:             "invalid prefix success",
			expectedResourceType: "",
		},
		{
			arn:                  "arn:not:enough:sections",
			caseName:             "invalid section structure, not enough sections success",
			expectedResourceType: "",
		},
		{
			arn:                  "arn:valid:example:with:6:sections",
			caseName:             "valid arn, structural example success",
			expectedResourceType: "",
		},
		{
			arn:                  "arn:aws:elasticbeanstalk:us-east-1:123456789012:environment/My App/MyEnvironment",
			caseName:             "valid arn, real example space in path success",
			expectedResourceType: "environment",
		},
		{
			arn:                  "arn:aws:iam::123456789012:user/David",
			caseName:             "valid arn, real example simple type and name success",
			expectedResourceType: "user",
		},
		{
			arn:                  "arn:aws:rds:eu-west-1:123456789012:db:mysql-db",
			caseName:             "valid arn, real example section separator path / more than usual section count success",
			expectedResourceType: "db",
		},
		{
			arn:                  "arn:aws:s3:::my_corporate_bucket/exampleobject.png",
			caseName:             "valid arn, real example no type success",
			expectedResourceType: "my_corporate_bucket",
		},
		{
			arn:                  "arn:valid:example:with:6:type/name:qualifier",
			caseName:             "valid arn, real example / type separator with qualifier success",
			expectedResourceType: "type",
		},
		{
			arn:                  "arn:valid:example:with:6:type:name:qualifier",
			caseName:             "valid arn, real example : type separator with qualifier success",
			expectedResourceType: "type",
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseName, func(t *testing.T) {
			actualResourceType := ResourceType(testCase.arn)

			require.Equal(t, testCase.expectedResourceType, actualResourceType)
		})
	}
}

func TestService(t *testing.T) {
	testCases := []struct {
		arn             string
		caseName        string
		expectedService string
	}{
		{
			arn:             "",
			caseName:        "empty string success",
			expectedService: "",
		},
		{
			arn:             "not-arn",
			caseName:        "invalid prefix success",
			expectedService: "",
		},
		{
			arn:             "arn:not:enough:sections",
			caseName:        "invalid section structure, not enough sections success",
			expectedService: "",
		},
		{
			arn:             "arn:valid:example:with:6:sections",
			caseName:        "valid arn, structural example success",
			expectedService: "example",
		},
		{
			arn:             "arn:aws:elasticbeanstalk:us-east-1:123456789012:environment/My App/MyEnvironment",
			caseName:        "valid arn, real example space in path success",
			expectedService: "elasticbeanstalk",
		},
		{
			arn:             "arn:aws:iam::123456789012:user/David",
			caseName:        "valid arn, real example simple type and name success",
			expectedService: "iam",
		},
		{
			arn:             "arn:aws:rds:eu-west-1:123456789012:db:mysql-db",
			caseName:        "valid arn, real example section separator path / more than usual section count success",
			expectedService: "rds",
		},
		{
			arn:             "arn:aws:s3:::my_corporate_bucket/exampleobject.png",
			caseName:        "valid arn, real example no type success",
			expectedService: "s3",
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseName, func(t *testing.T) {
			actualService := Service(testCase.arn)

			require.Equal(t, testCase.expectedService, actualService)
		})
	}
}

func TestValidateARN(t *testing.T) {
	testCases := []struct {
		caseName      string
		candidate     string
		expectedError error
	}{
		{
			caseName:      "empty string success",
			candidate:     "",
			expectedError: ErrorInvalidPrefix,
		},
		{
			caseName:      "invalid prefix success",
			candidate:     "not-arn",
			expectedError: ErrorInvalidPrefix,
		},
		{
			caseName:      "invalid section structure, not enough sections success",
			candidate:     "arn:not:enough:sections",
			expectedError: ErrorInvalidStructure,
		},
		{
			caseName:      "valid arn, structural example success",
			candidate:     "arn:valid:example:with:6:sections",
			expectedError: nil,
		},
		{
			caseName:      "valid arn, real example space in path success",
			candidate:     "arn:aws:elasticbeanstalk:us-east-1:123456789012:environment/My App/MyEnvironment",
			expectedError: nil,
		},
		{
			caseName:      "valid arn, real example simple type and name success",
			candidate:     "arn:aws:iam::123456789012:user/David",
			expectedError: nil,
		},
		{
			caseName:      "valid arn, real example section separator path / more than usual section count success",
			candidate:     "arn:aws:rds:eu-west-1:123456789012:db:mysql-db",
			expectedError: nil,
		},
		{
			caseName:      "valid arn, real example no type success",
			candidate:     "arn:aws:s3:::my_corporate_bucket/exampleobject.png",
			expectedError: nil,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseName, func(t *testing.T) {
			actualError := ValidateARN(testCase.candidate)

			if testCase.expectedError == nil {
				require.Nil(t, actualError)
			} else {
				require.EqualError(t, actualError, testCase.expectedError.Error())
			}
		})
	}
}
