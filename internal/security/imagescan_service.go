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

package anchore

import (
	"context"

	"emperror.dev/errors"

	"github.com/banzaicloud/pipeline/.gen/pipeline/pipeline"
	"github.com/banzaicloud/pipeline/internal/common"
)

// ImageScanner lists operations related to image scanning
type ImageScanner interface {
	// todo remove direct dependency on the generated types here

	// ScanImages adds the passed in images to be scanned by the underlying system (anchore)
	Scan(ctx context.Context, orgID uint, clusterID uint, images []pipeline.ClusterImage) (interface{}, error)

	// GetImageInfo retrieves the results of the scan for the given imageDigest
	GetImageInfo(ctx context.Context, orgID uint, clusterID uint, imageDigest string) (interface{}, error)

	// GetVulnerabilities retrieves the vulnerabilities for the given imageDigest
	GetVulnerabilities(ctx context.Context, orgID uint, clusterID uint, imageDigest string) (interface{}, error)
}

func NewImageScannerService(configService ConfigurationService, logger common.Logger) ImageScanner {
	return imageScannerService{
		configService: configService,
		logger:        logger,
	}
}

type imageScannerService struct {
	configService ConfigurationService
	logger        common.Logger
}

func (i imageScannerService) Scan(ctx context.Context, orgID uint, clusterID uint, images []pipeline.ClusterImage) (interface{}, error) {
	fnCtx := map[string]interface{}{"orgID": orgID, "clusterID": clusterID}
	i.logger.Info("scanning images", fnCtx)

	var (
		combinedErr error
		retImgs     = make([]interface{}, 0)
	)

	anchoreClient, err := i.getAnchoreClient(ctx, clusterID)
	if err != nil {
		return err, nil
	}

	for _, img := range images {
		// transform the input image
		img, err := anchoreClient.ScanImage(ctx, img)
		if err != nil {
			combinedErr = errors.Append(combinedErr, err)
			continue
		}

		retImgs = append(retImgs, img)
	}

	i.logger.Info("images sent for analysis", fnCtx)
	return retImgs, combinedErr
}

func (i imageScannerService) GetImageInfo(ctx context.Context, orgID uint, clusterID uint, imageDigest string) (interface{}, error) {
	fnCtx := map[string]interface{}{"orgID": orgID, "clusterID": clusterID, "imageDigest": imageDigest}
	i.logger.Info("getting scan results", fnCtx)

	anchoreClient, err := i.getAnchoreClient(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	imageInfo, err := anchoreClient.CheckImage(ctx, imageDigest)
	if err != nil {
		i.logger.Debug("failure while retrieving image information", fnCtx)

		return nil, errors.WrapIf(err, "failure while retrieving image information")
	}

	i.logger.Info("image info successfully retrieved", fnCtx)
	return imageInfo, nil
}

func (i imageScannerService) GetVulnerabilities(ctx context.Context, orgID uint, clusterID uint, imageDigest string) (interface{}, error) {
	fnCtx := map[string]interface{}{"orgID": orgID, "clusterID": clusterID, "imageDigest": imageDigest}
	i.logger.Info("retrieving image vulnerabilities", fnCtx)

	anchoreClient, err := i.getAnchoreClient(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	vulnerabilities, err := anchoreClient.GetImageVulnerabilities(ctx, imageDigest)
	if err != nil {
		i.logger.Debug("failure while retrieving image vulnerabilities", fnCtx)

		return nil, errors.WrapIf(err, "failure while retrieving image vulnerabilities")
	}

	i.logger.Info("vulnerabilities successfully retrieved", fnCtx)
	return vulnerabilities, nil
}

func (i imageScannerService) getAnchoreClient(ctx context.Context, clusterID uint) (AnchoreClient, error) {
	config, err := i.configService.GetConfiguration(ctx, clusterID)
	if err != nil {
		return nil, errors.WrapIfWithDetails(err, "failed to get anchore config for clster", "clusterID", clusterID)
	}

	return NewAnchoreClient(config, i.logger), nil
}
