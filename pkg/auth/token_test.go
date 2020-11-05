// Copyright © 2019 Banzai Cloud
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

package auth

import (
	"testing"
	"time"

	"emperror.dev/errors"
	"github.com/banzaicloud/bank-vaults/pkg/sdk/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type idGeneratorStub struct {
	id string
}

func (s idGeneratorStub) Generate() string {
	return s.id
}

type clockStub struct {
	now time.Time
}

func (s clockStub) Now() time.Time {
	return s.now
}

func newTime(value time.Time) *time.Time {
	return &value
}

func TestJWTTokenGenerator_GenerateToken(t *testing.T) {
	now := time.Date(2019, time.September, 20, 14, 44, 00, 00, time.UTC)

	generator := NewJWTTokenGenerator(
		"issuer",
		"audience",
		"signingKey",
		TokenIDGenerator(idGeneratorStub{"id"}),
		TokenGeneratorClock(clockStub{now}),
	)

	tokenID, signedToken, err := generator.GenerateToken("user", NoExpiration, "token", "my_text")
	require.NoError(t, err)

	const expectedSignedToken = "eyJhbGciOiJIUzI1NiJ9.eyJhdWQiOlsiYXVkaWVuY2UiXSwiaWF0IjoxNTY4OTkwNjQwLCJpc3MiOiJpc3N1ZXIiLCJqdGkiOiJpZCIsInNjb3BlIjoiYXBpOmludm9rZSIsInN1YiI6InVzZXIiLCJ0ZXh0IjoibXlfdGV4dCIsInR5cGUiOiJ0b2tlbiJ9.s1bPhRcl-tZpsxyFs9LACRrXVWwCmN5Q3PYH2EtscPc"

	assert.Equal(t, "id", tokenID)
	assert.Equal(t, expectedSignedToken, signedToken)
}

func TestTokenManager_GenerateToken(t *testing.T) {
	type inputType struct {
		m           TokenManager
		sub         string
		expiresAt   time.Time
		tokenType   TokenType
		tokenText   string
		tokenName   string
		storeSecret bool
	}

	type outputType struct {
		expectedTokenID     string
		expectedSignedToken string
		expectedError       error
		expectedStoredToken *auth.Token
	}

	testCases := []struct {
		caseDescription string
		input           inputType
		output          outputType
	}{
		{
			caseDescription: "token generator error -> error",
			input: inputType{
				m:           NewTokenManager(&MockTokenGenerator{}, auth.NewInMemoryTokenStore()),
				sub:         "subject",
				expiresAt:   NoExpiration,
				tokenType:   TokenType("token-type"),
				tokenText:   "token-text",
				tokenName:   "token-name",
				storeSecret: false,
			},
			output: outputType{
				expectedTokenID:     "",
				expectedSignedToken: "",
				expectedError:       errors.New("token generator error"),
				expectedStoredToken: nil,
			},
		},
		{
			caseDescription: "no expiration, no stored token -> success",
			input: inputType{
				m:           NewTokenManager(&MockTokenGenerator{}, auth.NewInMemoryTokenStore()),
				sub:         "subject",
				expiresAt:   NoExpiration,
				tokenType:   "token-type",
				tokenText:   "token-text",
				tokenName:   "token-name",
				storeSecret: false,
			},
			output: outputType{
				expectedTokenID:     "token-id",
				expectedSignedToken: "signed-token",
				expectedError:       nil,
				expectedStoredToken: &auth.Token{
					ID:        "token-id",
					Name:      "token-name",
					ExpiresAt: nil,
					CreatedAt: nil,
					Value:     "",
				},
			},
		},
		{
			caseDescription: "no expiration, stored token -> success",
			input: inputType{
				m:           NewTokenManager(&MockTokenGenerator{}, auth.NewInMemoryTokenStore()),
				sub:         "subject",
				expiresAt:   NoExpiration,
				tokenType:   "token-type",
				tokenText:   "token-text",
				tokenName:   "token-name",
				storeSecret: true,
			},
			output: outputType{
				expectedTokenID:     "token-id",
				expectedSignedToken: "signed-token",
				expectedError:       nil,
				expectedStoredToken: &auth.Token{
					ID:        "token-id",
					Name:      "token-name",
					ExpiresAt: nil,
					CreatedAt: nil,
					Value:     "signed-token",
				},
			},
		},
		{
			caseDescription: "expiration, no stored token -> success",
			input: inputType{
				m:           NewTokenManager(&MockTokenGenerator{}, auth.NewInMemoryTokenStore()),
				sub:         "subject",
				expiresAt:   time.Date(3020, 10, 5, 22, 31, 35, 9, time.UTC),
				tokenType:   "token-type",
				tokenText:   "token-text",
				tokenName:   "token-name",
				storeSecret: false,
			},
			output: outputType{
				expectedTokenID:     "token-id",
				expectedSignedToken: "signed-token",
				expectedError:       nil,
				expectedStoredToken: &auth.Token{
					ID:        "token-id",
					Name:      "token-name",
					ExpiresAt: newTime(time.Date(3020, 10, 5, 22, 31, 35, 9, time.UTC)),
					CreatedAt: nil,
					Value:     "",
				},
			},
		},
		{
			caseDescription: "expiration, stored token -> success",
			input: inputType{
				m:           NewTokenManager(&MockTokenGenerator{}, auth.NewInMemoryTokenStore()),
				sub:         "subject",
				expiresAt:   time.Date(3020, 10, 5, 22, 31, 35, 9, time.UTC),
				tokenType:   "token-type",
				tokenText:   "token-text",
				tokenName:   "token-name",
				storeSecret: true,
			},
			output: outputType{
				expectedTokenID:     "token-id",
				expectedSignedToken: "signed-token",
				expectedError:       nil,
				expectedStoredToken: &auth.Token{
					ID:        "token-id",
					Name:      "token-name",
					ExpiresAt: newTime(time.Date(3020, 10, 5, 22, 31, 35, 9, time.UTC)),
					CreatedAt: nil,
					Value:     "signed-token",
				},
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			mockTokenGenerator := testCase.input.m.generator.(*MockTokenGenerator)
			generateTokenMock := mockTokenGenerator.On(
				"GenerateToken",
				testCase.input.sub,
				testCase.input.expiresAt,
				string(testCase.input.tokenType),
				testCase.input.tokenText,
			)

			if testCase.output.expectedError != nil &&
				testCase.output.expectedError.Error() == "token generator error" {
				generateTokenMock.Return(
					"",
					"",
					testCase.output.expectedError,
				).Once()
			} else {
				generateTokenMock.Return(
					testCase.output.expectedTokenID,
					testCase.output.expectedSignedToken,
					nil,
				).Once()
			}

			actualTokenID, actualSignedToken, actualError := testCase.input.m.GenerateToken(
				testCase.input.sub,
				testCase.input.expiresAt,
				testCase.input.tokenType,
				testCase.input.tokenText,
				testCase.input.tokenName,
				testCase.input.storeSecret,
			)

			if testCase.output.expectedError == nil {
				require.NoError(t, actualError)
			} else {
				require.EqualError(t, actualError, testCase.output.expectedError.Error())
			}
			require.Equal(t, testCase.output.expectedTokenID, actualTokenID)
			require.Equal(t, testCase.output.expectedSignedToken, actualSignedToken)

			if testCase.output.expectedError == nil {
				actualStoredToken, err := testCase.input.m.store.Lookup(
					testCase.input.sub,
					testCase.output.expectedTokenID,
				)
				require.NoError(t, err)
				require.NotNil(t, actualStoredToken)

				// Note: faking dynamically generated values.
				if testCase.output.expectedStoredToken.CreatedAt == nil {
					require.Nil(t, actualStoredToken.CreatedAt)
				} else {
					require.NotNil(t, actualStoredToken.CreatedAt)
					require.InEpsilon(
						t,
						testCase.output.expectedStoredToken.CreatedAt.Unix(),
						actualStoredToken.CreatedAt.Unix(),
						3.0,
						"creation time is not within threshold",
					)
					actualStoredToken.CreatedAt = testCase.output.expectedStoredToken.CreatedAt
				}

				require.Equal(t, testCase.output.expectedStoredToken, actualStoredToken)
			}

			mockTokenGenerator.AssertExpectations(t)
		})
	}
}
