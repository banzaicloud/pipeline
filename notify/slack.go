package notify

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"github.com/banzaicloud/pipeline/conf"
	"fmt"
)

type Slack struct {
	Text       string `json:"text"`
	Username   string `json:"username"`
	Icon_emoji string `json:"icon_emoji"`
	Icon_url   string `json:"icon_url"`
	Channel    string `json:"channel"`
}

func SlackNotify(message string) error{

	log := conf.Logger()

	content := Slack{}

	if len(os.Getenv("SLACK_WEBHOOK_URL")) <= 0 {
		log.Infoln("Webhookurl is missing -> Slack notification disabled.")
		return nil
	}
	webhookUrl := os.Getenv("SLACK_WEBHOOK_URL")


	if len(os.Getenv("SLACK_CHANNEL")) <= 0 {
		log.Infoln("Channel name is missing -> Slack notification disabled.")
		return nil
	}
	content.Channel = os.Getenv("SLACK_CHANNEL")


	content.Icon_emoji = ":cloud:"
	content.Username = "Pipeline"
	content.Text = message

	params, marsErr := json.Marshal(content)
	if marsErr != nil {
		log.Warningln(marsErr)
		return fmt.Errorf("Content masrshalling failed: %s" , marsErr)
	}


	resp, postErr := http.PostForm(webhookUrl, url.Values{"payload": {string(params)}})
	if postErr != nil {
		log.Warningln("Slack API Post Failed:", postErr)
		return fmt.Errorf("http post form failed: %s" , postErr)
	}

	body, respErr := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if respErr != nil {
		log.Warningln("Slack Response Failed:", respErr)
		return fmt.Errorf("http response failed: %s" , respErr)
	}
	log.Debugln("Slack API Response:",string(body))
	return nil
}
