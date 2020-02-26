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

package oci

// GetSupportedShapes gives back supported node shapes in all subscribed regions
func (oci *OCI) GetSupportedShapes() (shapes map[string][]string, err error) {
	ic, err := oci.NewIdentityClient()
	if err != nil {
		return shapes, err
	}

	regions, err := ic.GetSubscribedRegionNames()
	if err != nil {
		return shapes, err
	}

	shapes = make(map[string][]string, 0)
	for _, region := range regions {
		_shapes, err := oci.GetSupportedShapesInARegion(region)
		if err != nil {
			return shapes, err
		}
		shapes[region] = _shapes
	}

	return shapes, err
}

// GetSupportedShapesInARegion gives back supported node shapes in the given region
func (oci *OCI) GetSupportedShapesInARegion(region string) (shapes []string, err error) {
	err = oci.ChangeRegion(region)
	if err != nil {
		return shapes, err
	}

	ce, err := oci.NewContainerEngineClient()
	if err != nil {
		return nil, err
	}

	options, err := ce.GetDefaultNodePoolOptions()
	if err != nil {
		return nil, err
	}

	return options.Shapes.Get(), nil
}

// GetSupportedImages gives back supported node images in all subscribed regions
func (oci *OCI) GetSupportedImages() (images map[string][]string, err error) {
	ic, err := oci.NewIdentityClient()
	if err != nil {
		return images, err
	}

	regions, err := ic.GetSubscribedRegionNames()
	if err != nil {
		return images, err
	}

	images = make(map[string][]string, 0)
	for _, region := range regions {
		_images, err := oci.GetSupportedImagesInARegion(region)
		if err != nil {
			return images, err
		}
		images[region] = _images
	}

	return images, err
}

// GetSupportedImagesInARegion gives back supported node images in the given region
func (oci *OCI) GetSupportedImagesInARegion(region string) (images []string, err error) {
	err = oci.ChangeRegion(region)
	if err != nil {
		return nil, err
	}

	ce, err := oci.NewContainerEngineClient()
	if err != nil {
		return nil, err
	}

	options, err := ce.GetDefaultNodePoolOptions()
	if err != nil {
		return nil, err
	}

	return options.Images.Get(), nil
}

// GetSupportedK8SVersions gives back supported k8s versions in all subscribed regions
func (oci *OCI) GetSupportedK8SVersions() (k8sversions map[string][]string, err error) {
	ic, err := oci.NewIdentityClient()
	if err != nil {
		return k8sversions, err
	}

	regions, err := ic.GetSubscribedRegionNames()
	if err != nil {
		return k8sversions, err
	}

	k8sversions = make(map[string][]string, 0)
	for _, region := range regions {
		_k8sversions, err := oci.GetSupportedK8SVersionsInARegion(region)
		if err != nil {
			return k8sversions, err
		}
		k8sversions[region] = _k8sversions
	}

	return k8sversions, err
}

// GetSupportedK8SVersionsInARegion gives back supported k8s versions in the given region
func (oci *OCI) GetSupportedK8SVersionsInARegion(region string) (k8sversions []string, err error) {
	err = oci.ChangeRegion(region)
	if err != nil {
		return nil, err
	}

	ce, err := oci.NewContainerEngineClient()
	if err != nil {
		return nil, err
	}

	options, err := ce.GetDefaultNodePoolOptions()
	if err != nil {
		return nil, err
	}

	return options.KubernetesVersions.Get(), nil
}
