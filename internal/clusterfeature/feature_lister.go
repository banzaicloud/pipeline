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

package clusterfeature

import (
	"context"

	"emperror.dev/errors"
	"github.com/goph/logur"
)

type featureLister struct {
	logger            logur.Logger
	featureRepository FeatureRepository
}

func (fl *featureLister) List(ctx context.Context, clusterID uint) ([]Feature, error) {

	mLogger := logur.WithFields(fl.logger, map[string]interface{}{"clusterId": clusterID})
	mLogger.Debug("retrieving features ...")

	features, err := fl.featureRepository.ListFeatures(ctx, clusterID)
	if err != nil {
		mLogger.Debug("failed to retrieve features")

		return nil, errors.WrapIf(err, "failed to retrieve features")
	}

	mLogger.Debug("features successfully retrieved")
	return features, nil

}

func NewFeatureLister(logger logur.Logger, featureRepository FeatureRepository) FeatureLister {
	return &featureLister{
		logger:            logur.WithFields(logger, map[string]interface{}{"feature-lister": "comp"}),
		featureRepository: featureRepository,
	}
}
