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

package cloudformation

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"text/template"

	"emperror.dev/errors"

	"github.com/banzaicloud/pipeline/internal/global"
)

// GetCloudFormationTemplate returns CloudFormation template with given name
func GetCloudFormationTemplate(path, name string) (string, error) {
	templatePath := path + "/" + name
	u, err := url.Parse(templatePath)
	if err != nil {
		return "", errors.WrapIf(err, fmt.Sprintf("failed to read CloudFormation template from %s", templatePath))
	}

	var content []byte
	if u.Scheme == "file" || u.Scheme == "" {
		content, err = ioutil.ReadFile(templatePath)
	} else if u.Scheme == "http" || u.Scheme == "https" {
		var resp *http.Response
		resp, err = http.Get(u.String())
		if err == nil {
			content, err = ioutil.ReadAll(resp.Body)
			defer resp.Body.Close()
		}
	} else {
		err = fmt.Errorf("not supported scheme: %s", u.Scheme)
	}

	if err != nil {
		return "", errors.WrapIf(err, fmt.Sprintf("failed to read CloudFormation template content from %s", templatePath))
	}

	t, err := template.New("cf").Parse(string(content))
	if err != nil {
		return "", errors.WrapIf(err, fmt.Sprintf("failed to parse CloudFormation template content from %s", templatePath))
	}

	buffer := bytes.NewBuffer(nil)
	data := map[string]interface{}{
		"UpdatePolicyEnabled": !global.Config.Pipeline.Enterprise,
	}
	err = t.Execute(buffer, data)
	if err != nil {
		return "", errors.WrapIf(err, fmt.Sprintf("failed to evaluate CloudFormation template content from %s", templatePath))
	}

	return buffer.String(), nil
}
