// Copyright © 2018 Banzai Cloud
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

package api

import (
	"fmt"
	"net/http"

	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/validation"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/banzaicloud/pipeline/internal/objectstore"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	pkgErrors "github.com/banzaicloud/pipeline/pkg/errors"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/googleapi"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// listPods returns list of pods in all namespaces.
func listPods(client *kubernetes.Clientset, fieldSelector string, labelSelector string) ([]v1.Pod, error) {
	log := log.WithFields(logrus.Fields{
		"fieldSelector": fieldSelector,
		"labelSelector": labelSelector,
	})

	log.Debug("List pods")
	podList, err := client.CoreV1().Pods(metav1.NamespaceAll).List(metav1.ListOptions{
		FieldSelector: fieldSelector,
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, err
	}

	return podList.Items, nil
}

// ErrorResponseFrom translates the given error into a components.ErrorResponse
func ErrorResponseFrom(err error) *pkgCommon.ErrorResponse {
	if objectstore.IsNotFoundError(err) {
		return &pkgCommon.ErrorResponse{
			Code:    http.StatusNotFound,
			Error:   err.Error(),
			Message: err.Error(),
		}
	}

	// google specific errors
	if googleApiErr, ok := err.(*googleapi.Error); ok {
		return &pkgCommon.ErrorResponse{
			Code:    googleApiErr.Code,
			Error:   googleApiErr.Error(),
			Message: googleApiErr.Message,
		}
	}

	// aws specific errors
	if awsErr, ok := err.(awserr.Error); ok {
		code := http.StatusBadRequest
		if awsReqFailure, ok := err.(awserr.RequestFailure); ok {
			code = awsReqFailure.StatusCode()
		}

		return &pkgCommon.ErrorResponse{
			Code:    code,
			Error:   awsErr.Error(),
			Message: awsErr.Message(),
		}
	}

	// azure specific errors
	if azureErr, ok := err.(validation.Error); ok {
		return &pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Error:   azureErr.Error(),
			Message: azureErr.Message,
		}
	}

	if azureErr, ok := err.(azblob.StorageError); ok {
		serviceCode := fmt.Sprint(azureErr.ServiceCode())

		return &pkgCommon.ErrorResponse{
			Code:    azureErr.Response().StatusCode,
			Error:   azureErr.Error(),
			Message: serviceCode,
		}
	}

	if azureErr, ok := err.(autorest.DetailedError); ok {
		if azureErr.Original != nil {
			if azureOrigErr, ok := azureErr.Original.(*azure.RequestError); ok {
				return &pkgCommon.ErrorResponse{
					Code:    azureErr.Response.StatusCode,
					Error:   azureOrigErr.ServiceError.Error(),
					Message: azureOrigErr.ServiceError.Message,
				}
			}

			return &pkgCommon.ErrorResponse{
				Code:    azureErr.Response.StatusCode,
				Error:   azureErr.Original.Error(),
				Message: azureErr.Message,
			}
		}

		return &pkgCommon.ErrorResponse{
			Code:    azureErr.Response.StatusCode,
			Error:   azureErr.Error(),
			Message: azureErr.Message,
		}
	}

	// pipeline specific errors
	if err == pkgErrors.ErrorNotSupportedCloudType {
		return &pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Error:   err.Error(),
			Message: err.Error(),
		}
	}

	if errors.Cause(err) == pkgErrors.ErrorBucketDeleteNotEmpty {
		return &pkgCommon.ErrorResponse{
			Code:    http.StatusConflict,
			Error:   err.Error(),
			Message: err.Error(),
		}
	}

	switch err.(type) {
	case SecretNotFoundError, secret.MismatchError:
		return &pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Error:   err.Error(),
			Message: err.Error(),
		}
	default:
		return &pkgCommon.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Error:   err.Error(),
			Message: err.Error(),
		}
	}
}
