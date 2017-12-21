package notify

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	banzaiUtils "github.com/banzaicloud/banzai-types/utils"
	banzaiConstants "github.com/banzaicloud/banzai-types/constants"
)

//Slack message definition
type Slack struct {
	Text      string `json:"text"`
	Username  string `json:"username"`
	iconEmoji string `json:"iconemoji"`
	iconUrl   string `json:"iconurl"`
	Channel   string `json:"channel"`
}

//SlackNotify is pushing to Slack
func SlackNotify(message string) error {

	content := Slack{}

	if len(os.Getenv("SLACK_WEBHOOK_URL")) <= 0 {
		banzaiUtils.LogInfo(banzaiConstants.TagSlack, "Webhookurl is missing -> Slack notification disabled.")
		return nil
	}
	webhookUrl := os.Getenv("SLACK_WEBHOOK_URL")

	if len(os.Getenv("SLACK_CHANNEL")) <= 0 {
		banzaiUtils.LogInfo(banzaiConstants.TagSlack, "Channel name is missing -> Slack notification disabled.")
		return nil
	}
	content.Channel = os.Getenv("SLACK_CHANNEL")

	content.iconEmoji = ":cloud:"
	content.Username = "banzaicloud"
	content.Text = message

	params, marsErr := json.Marshal(content)
	if marsErr != nil {
		banzaiUtils.LogWarn(banzaiConstants.TagSlack, marsErr)
		return fmt.Errorf("Content masrshalling failed: %s", marsErr)
	}

	resp, postErr := http.PostForm(webhookUrl, url.Values{"payload": {string(params)}})
	if postErr != nil {
		banzaiUtils.LogWarn(banzaiConstants.TagSlack, "Slack API Post Failed:", postErr)
		return fmt.Errorf("http post form failed: %s", postErr)
	}

	body, respErr := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if respErr != nil {
		banzaiUtils.LogWarn(banzaiConstants.TagSlack, "Slack Response Failed:", respErr)
		return fmt.Errorf("http response failed: %s", respErr)
	}
	banzaiUtils.LogDebug(banzaiConstants.TagSlack, "Slack API Response:", string(body))
	return nil
}
