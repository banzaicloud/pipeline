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

package integratedservices

import (
	"emperror.dev/errors"
)

// MakeIntegratedServiceManagerRegistry returns a IntegratedServiceManagerRegistry with the specified integrated service managers registered.
func MakeIntegratedServiceManagerRegistry(managers []IntegratedServiceManager) IntegratedServiceManagerRegistry {
	lookup := make(map[string]IntegratedServiceManager, len(managers))
	for _, fm := range managers {
		lookup[fm.Name()] = fm
	}

	return integratedServiceManagerRegistry{
		lookup: lookup,
	}
}

type integratedServiceManagerRegistry struct {
	lookup map[string]IntegratedServiceManager
}

func (r integratedServiceManagerRegistry) GetIntegratedServiceManager(integratedServiceName string) (IntegratedServiceManager, error) {
	if integratedServiceManager, ok := r.lookup[integratedServiceName]; ok {
		return integratedServiceManager, nil
	}

	return nil, errors.WithStack(UnknownIntegratedServiceError{IntegratedServiceName: integratedServiceName})
}

func (r integratedServiceManagerRegistry) GetIntegratedServiceNames() []string {
	keys := make([]string, 0)
	for key := range r.lookup {
		keys = append(keys, key)
	}
	return keys
}

// MakeIntegratedServiceOperatorRegistry returns a IntegratedServiceOperatorRegistry with the specified integrated service operators registered.
func MakeIntegratedServiceOperatorRegistry(operators []IntegratedServiceOperator) IntegratedServiceOperatorRegistry {
	lookup := make(map[string]IntegratedServiceOperator, len(operators))
	for _, fo := range operators {
		lookup[fo.Name()] = fo
	}

	return integratedServiceOperatorRegistry{
		lookup: lookup,
	}
}

type integratedServiceOperatorRegistry struct {
	lookup map[string]IntegratedServiceOperator
}

func (r integratedServiceOperatorRegistry) GetIntegratedServiceOperator(integratedServiceName string) (IntegratedServiceOperator, error) {
	if integratedServiceOperator, ok := r.lookup[integratedServiceName]; ok {
		return integratedServiceOperator, nil
	}

	return nil, errors.WithStack(UnknownIntegratedServiceError{IntegratedServiceName: integratedServiceName})
}

// UnknownIntegratedServiceError is returned when there is no integrated service manager registered for a integrated service.
type UnknownIntegratedServiceError struct {
	IntegratedServiceName string
}

func (UnknownIntegratedServiceError) Error() string {
	return "unknown integrated service"
}

// Details returns the error's details
func (e UnknownIntegratedServiceError) Details() []interface{} {
	return []interface{}{"integratedService", e.IntegratedServiceName}
}

// ServiceError tells the transport layer whether this error should be translated into the transport format
// or an internal error should be returned instead.
func (UnknownIntegratedServiceError) ServiceError() bool {
	return true
}

// Unknown tells a client that this error is related to a resource being unsupported.
// Can be used to translate the error to eg. status code.
func (UnknownIntegratedServiceError) Unknown() bool {
	return true
}
