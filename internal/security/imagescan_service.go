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

	"github.com/banzaicloud/pipeline/internal/common"
)

type ImageData struct {
	Tag    string `json:"tag,omitempty"`
	Digest string `json:"digest,omitempty"`
}

// ImageScanner lists operations related to image scanning
type ImageScanner interface {
	ScanImages(ctx context.Context, orgID uint, clusterID uint, images []ImageData) error
	GetScanResults(ctx context.Context)
	GetImageVulnerabilities(ctx context.Context)
}

func MakeImageScannerService(configService ConfigurationService, logger common.Logger) ImageScanner {
	return imageScannerService{
		configService: configService,
		logger:        logger,
	}
}

type imageScannerService struct {
	configService ConfigurationService
	logger        common.Logger
}

func (i imageScannerService) ScanImages(ctx context.Context, orgID uint, clusterID uint, images []ImageData) error {
	fnCtx := map[string]interface{}{"orgID": orgID, "clusterID": clusterID}
	i.logger.Info("scanning images", fnCtx)

	var combinedErr error

	anchoreCli, err := i.getAnchoreClient(ctx, clusterID)
	if err != nil {
		return err
	}

	for _, img := range images {
		// transform the input image
		err := anchoreCli.ScanImage(ctx, img)
		combinedErr = errors.Append(combinedErr, err)
	}

	i.logger.Info("images sent for analysis", fnCtx)
	return combinedErr
}

func (i imageScannerService) GetScanResults(ctx context.Context) {
	panic("implement me")
}

func (i imageScannerService) GetImageVulnerabilities(ctx context.Context) {
	panic("implement me")
}

func (i imageScannerService) getAnchoreClient(ctx context.Context, clusterID uint) (AnchoreClient, error) {
	config, err := i.configService.GetConfiguration(ctx, clusterID)
	if err != nil {
		return nil, errors.WrapIfWithDetails(err, "failed to get anchore config for clster", "clusterID", clusterID)
	}

	return MakeAnchoreClient(config, i.logger), nil
}
