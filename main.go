package main

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/opsgenie/opsgenie-go-sdk-v2/client"
	"github.com/opsgenie/opsgenie-go-sdk-v2/schedule"
	"github.com/opsgenie/opsgenie-go-sdk-v2/user"
	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"
)

type OncallRotation struct {
	Primary    string
	Secondary  string
	Notes      string
	LastUpdate string
}

func main() {
	log.Println("Starting oncall notifier...")

	if os.Getenv("OPSGENIE_APIKEY") == "" {
		log.Fatal("missing Opsgenie api key")
		return
	}

	if os.Getenv("MATTERMOST_HOOK") == "" {
		log.Fatal("missing mattermost webhook")
		return
	}

	tmpl := template.Must(template.ParseFiles("assets/oncall.html"))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		primary, secondary := whoIsOnCall("github_username")
		notes := ""
		if primary == "" {
			primary = "jwilander"
			notes = "No Engineer is oncall right now, if any incident happens it will escalate directly to the Engineer Manager"
			secondary = ""
		}
		data := OncallRotation{
			Primary:    primary,
			Secondary:  secondary,
			Notes:      notes,
			LastUpdate: time.Now().Local().Format("Mon Jan 2 15:04:05 -0700 MST 2006"),
		}
		tmpl.Execute(w, data)
	})

	c := cron.New()
	c.AddFunc("30 8,15 * * 1-5", sendMattermostWhoisOnCall)

	go c.Start()
	go http.ListenAndServe(":8077", nil)

	sig := make(chan os.Signal)
	signal.Notify(sig, os.Interrupt, os.Kill)
	signal.Notify(sig, syscall.SIGTERM, os.Kill)
	<-sig

}

func whoIsOnCall(userNameType string) (string, string) {

	primary, err := getOncall("BizOps_schedule", userNameType)
	if err != nil {
		log.Println("not able to get who is the primary")
		return "", ""
	}

	if primary == "" {
		return "", ""
	}

	secondary, err := getOncall("Backup Cloud_schedule", userNameType)
	if err != nil {
		log.Fatal("not able to get who is the secondary")
		return "", ""
	}

	return primary, secondary
}

func sendMattermostWhoisOnCall() {
	primary, secondary := whoIsOnCall("mattermost_username")
	sendWhoIsOnCallNotification(primary, secondary)
}

func getUserInfo(opsGenieUser, userNameType string) string {
	client, err := user.NewClient(&client.Config{
		ApiKey: os.Getenv("OPSGENIE_APIKEY"),
	})
	if err != nil {
		log.Fatal("not able to create a new opsgenie client")
		return ""
	}

	userReq := &user.GetRequest{
		Identifier: opsGenieUser,
	}

	userResult, err := client.Get(context.Background(), userReq)
	if err != nil {
		log.Fatal("not able to get the user")
		return ""
	}

	log.Println(userResult.FullName, userResult.Username)
	if len(userResult.Details[userNameType]) == 0 {
		return userResult.FullName
	}

	return userResult.Details[userNameType][0]
}

func getOncall(scheduleName, userNameType string) (string, error) {
	client, err := schedule.NewClient(&client.Config{
		ApiKey:   os.Getenv("OPSGENIE_APIKEY"),
		LogLevel: logrus.DebugLevel,
	})
	if err != nil {
		log.Fatal("not able to create a new opsgenie client")
		return "", err
	}

	flat := true
	now := time.Now()
	onCallReq := &schedule.GetOnCallsRequest{
		Flat:                   &flat,
		Date:                   &now,
		ScheduleIdentifierType: schedule.Name,
		ScheduleIdentifier:     scheduleName,
	}
	onCall, err := client.GetOnCalls(context.TODO(), onCallReq)
	if err != nil {
		log.Fatal("not able to get who is on call 2")
		return "", err
	}

	if (len(onCall.OnCallRecipients)) <= 0 {
		return "", nil
	}

	fmt.Println("<>>>>>", onCall.OnCallRecipients[0])

	primary := getUserInfo(onCall.OnCallRecipients[0], userNameType)
	if primary == "" {
		return onCall.OnCallRecipients[0], nil
	}

	return primary, nil
}
