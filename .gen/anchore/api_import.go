/*
Anchore Engine API Server

This is the Anchore Engine API. Provides the primary external API for users of the service.

API version: 0.1.20
Contact: nurmi@anchore.com
*/

// Code generated by OpenAPI Generator (https://openapi-generator.tech); DO NOT EDIT.

package anchore

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
)


// ImportApiService ImportApi service
type ImportApiService service

type ApiImportImageArchiveRequest struct {
	ctx context.Context
	ApiService *ImportApiService
	archiveFile **os.File
}

// anchore image tar archive.
func (r ApiImportImageArchiveRequest) ArchiveFile(archiveFile *os.File) ApiImportImageArchiveRequest {
	r.archiveFile = &archiveFile
	return r
}

func (r ApiImportImageArchiveRequest) Execute() ([]AnchoreImage, *http.Response, error) {
	return r.ApiService.ImportImageArchiveExecute(r)
}

/*
ImportImageArchive Import an anchore image tar.gz archive file. This is a deprecated API replaced by the \"/imports/images\" route

 @param ctx context.Context - for authentication, logging, cancellation, deadlines, tracing, etc. Passed from http.Request or context.Background().
 @return ApiImportImageArchiveRequest
*/
func (a *ImportApiService) ImportImageArchive(ctx context.Context) ApiImportImageArchiveRequest {
	return ApiImportImageArchiveRequest{
		ApiService: a,
		ctx: ctx,
	}
}

// Execute executes the request
//  @return []AnchoreImage
func (a *ImportApiService) ImportImageArchiveExecute(r ApiImportImageArchiveRequest) ([]AnchoreImage, *http.Response, error) {
	var (
		localVarHTTPMethod   = http.MethodPost
		localVarPostBody     interface{}
		formFiles            []formFile
		localVarReturnValue  []AnchoreImage
	)

	localBasePath, err := a.client.cfg.ServerURLWithContext(r.ctx, "ImportApiService.ImportImageArchive")
	if err != nil {
		return localVarReturnValue, nil, &GenericOpenAPIError{error: err.Error()}
	}

	localVarPath := localBasePath + "/import/images"

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := url.Values{}
	localVarFormParams := url.Values{}
	if r.archiveFile == nil {
		return localVarReturnValue, nil, reportError("archiveFile is required and must be specified")
	}

	// to determine the Content-Type header
	localVarHTTPContentTypes := []string{"multipart/form-data"}

	// set Content-Type header
	localVarHTTPContentType := selectHeaderContentType(localVarHTTPContentTypes)
	if localVarHTTPContentType != "" {
		localVarHeaderParams["Content-Type"] = localVarHTTPContentType
	}

	// to determine the Accept header
	localVarHTTPHeaderAccepts := []string{"application/json"}

	// set Accept header
	localVarHTTPHeaderAccept := selectHeaderAccept(localVarHTTPHeaderAccepts)
	if localVarHTTPHeaderAccept != "" {
		localVarHeaderParams["Accept"] = localVarHTTPHeaderAccept
	}
	var archiveFileLocalVarFormFileName string
	var archiveFileLocalVarFileName     string
	var archiveFileLocalVarFileBytes    []byte

	archiveFileLocalVarFormFileName = "archive_file"

	archiveFileLocalVarFile := *r.archiveFile
	if archiveFileLocalVarFile != nil {
		fbs, _ := ioutil.ReadAll(archiveFileLocalVarFile)
		archiveFileLocalVarFileBytes = fbs
		archiveFileLocalVarFileName = archiveFileLocalVarFile.Name()
		archiveFileLocalVarFile.Close()
	}
	formFiles = append(formFiles, formFile{fileBytes: archiveFileLocalVarFileBytes, fileName: archiveFileLocalVarFileName, formFileName: archiveFileLocalVarFormFileName})
	req, err := a.client.prepareRequest(r.ctx, localVarPath, localVarHTTPMethod, localVarPostBody, localVarHeaderParams, localVarQueryParams, localVarFormParams, formFiles)
	if err != nil {
		return localVarReturnValue, nil, err
	}

	localVarHTTPResponse, err := a.client.callAPI(req)
	if err != nil || localVarHTTPResponse == nil {
		return localVarReturnValue, localVarHTTPResponse, err
	}

	localVarBody, err := ioutil.ReadAll(localVarHTTPResponse.Body)
	localVarHTTPResponse.Body.Close()
	localVarHTTPResponse.Body = ioutil.NopCloser(bytes.NewBuffer(localVarBody))
	if err != nil {
		return localVarReturnValue, localVarHTTPResponse, err
	}

	if localVarHTTPResponse.StatusCode >= 300 {
		newErr := &GenericOpenAPIError{
			body:  localVarBody,
			error: localVarHTTPResponse.Status,
		}
		if localVarHTTPResponse.StatusCode == 500 {
			var v ApiErrorResponse
			err = a.client.decode(&v, localVarBody, localVarHTTPResponse.Header.Get("Content-Type"))
			if err != nil {
				newErr.error = err.Error()
				return localVarReturnValue, localVarHTTPResponse, newErr
			}
			newErr.model = v
		}
		return localVarReturnValue, localVarHTTPResponse, newErr
	}

	err = a.client.decode(&localVarReturnValue, localVarBody, localVarHTTPResponse.Header.Get("Content-Type"))
	if err != nil {
		newErr := &GenericOpenAPIError{
			body:  localVarBody,
			error: err.Error(),
		}
		return localVarReturnValue, localVarHTTPResponse, newErr
	}

	return localVarReturnValue, localVarHTTPResponse, nil
}
