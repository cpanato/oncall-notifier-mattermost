package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"

	model "github.com/mattermost/mattermost-server/v5/model"
	"github.com/pkg/errors"
)

func send(webhookURL string, payload model.CommandResponse) error {
	marshalContent, _ := json.Marshal(payload)
	var jsonStr = []byte(marshalContent)
	req, err := http.NewRequest("POST", webhookURL, bytes.NewBuffer(jsonStr))
	req.Header.Set("X-Custom-Header", "aws-sns")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return errors.Wrap(err, "failed tο send HTTP request")
	}
	defer resp.Body.Close()
	return nil
}

func sendWhoIsOnCallNotification(primary, secondary string) error {
	attachment := &model.SlackAttachment{
		Title:     "OpsGenie",
		TitleLink: "https://mattermost.app.opsgenie.com/alert",
		Text:      "You also can check any time in this [page](https://cloud-oncall.core.cloud.mattermost.com/)",
		Color:     "#0000ff",
		Fields: []*model.SlackAttachmentField{
			{Title: "Who is onCall?", Short: false},
			{Title: "Primary", Value: primary, Short: true},
			{Title: "Secondary", Value: secondary, Short: true},
			{Value: "_Who you gonna call?_", Short: false},
		},
	}

	payload := model.CommandResponse{
		Username:    "OnCall Notifier",
		IconURL:     "https://vignette.wikia.nocookie.net/ghostbusters/images/a/a7/NoGhostSign.jpg/revision/latest/scale-to-width-down/340?cb=20090213041921",
		Attachments: []*model.SlackAttachment{attachment},
	}
	err := send(os.Getenv("MATTERMOST_HOOK"), payload)
	if err != nil {
		return errors.Wrap(err, "failed tο send Mattermost request payload")
	}
	return nil
}
