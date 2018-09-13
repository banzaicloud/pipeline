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

package notify

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
)

//Slack message definition
type Slack struct {
	Text      string `json:"text"`
	Username  string `json:"username"`
	IconEmoji string `json:"iconemoji"`
	IconUrl   string `json:"iconurl"`
	Channel   string `json:"channel"`
}

//SlackNotify is pushing to Slack
func SlackNotify(message string) error {
	content := Slack{}

	if len(os.Getenv("SLACK_WEBHOOK_URL")) <= 0 {
		log.Info("Webhookurl is missing -> Slack notification disabled.")
		return nil
	}
	webhookUrl := os.Getenv("SLACK_WEBHOOK_URL")

	if len(os.Getenv("SLACK_CHANNEL")) <= 0 {
		log.Info("Channel name is missing -> Slack notification disabled.")
		return nil
	}
	content.Channel = os.Getenv("SLACK_CHANNEL")

	content.IconEmoji = ":cloud:"
	content.Username = "banzaicloud"
	content.Text = message

	params, marsErr := json.Marshal(content)
	if marsErr != nil {
		log.Debug(marsErr)
		return fmt.Errorf("Content masrshalling failed: %s", marsErr)
	}

	resp, postErr := http.PostForm(webhookUrl, url.Values{"payload": {string(params)}})
	if postErr != nil {
		log.Debug("Slack API Post Failed:", postErr)
		return fmt.Errorf("http post form failed: %s", postErr)
	}

	body, respErr := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if respErr != nil {
		log.Debug("Slack Response Failed:", respErr)
		return fmt.Errorf("http response failed: %s", respErr)
	}
	log.Debug("Slack API Response:", string(body))
	return nil
}
