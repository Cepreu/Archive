package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/WorkFit/commongo/polling"
	"github.com/WorkFit/go/aws/sqs"
	"github.com/WorkFit/go/caldav"
	"github.com/WorkFit/go/calendar"
	"github.com/WorkFit/go/ews"
	"github.com/WorkFit/go/google"
	"github.com/WorkFit/go/log"
	"github.com/WorkFit/go/parse"
	"github.com/WorkFit/go/secrets"
)

var (
	queue      sqs.MessageQueue
	debugUsers = os.Getenv("DEBUG_USERS")
	queueURL   = os.Getenv("USER_OBJECTS_QUEUE_URL")
)

func main() {
	defer logBeforeExiting()

	flag.Parse()
	queue = sqs.NewMessageQueue(queueURL)
	poller := polling.NewBernoulliExponentialBackoffPoller(queue, 0.95, time.Millisecond, time.Minute)
	go poller.Start()
	go consumeMessages(poller.Channel())
	waitIndefinitely()
}

func consumeMessages(channel <-chan interface{}) {
	log.Debug("Started consuming messages")
	for batch := range channel {
		messages := batch.([]*sqs.Message)
		deleteMessages(messages)
		log.Debug("Received messages", "len(messages)", len(messages))
		for _, message := range messages {
			go logNonNilError(processMessage(message))
		}
	}
}

func deleteMessages(messages []*sqs.Message) {
	handles := make([]string, len(messages))
	for i, message := range messages {
		handles[i] = message.Handle
	}

	log.Debug("Deleting messages", "handles", handles)
	logNonNilError(queue.DeleteMessages(handles))
}

func processMessage(message *sqs.Message) error {
	log.Debug("Processing message", "message", message.Body)

	payload := map[string]string{}
	err := json.Unmarshal([]byte(message.Body), &payload)
	if err != nil {
		return err
	}

	user := &user{}
	body := strings.Replace(payload["Message"], "\\\"", "\"", -1)
	err = json.Unmarshal([]byte(body), user)
	if err != nil {
		return err
	}

	if strings.Contains(debugUsers, user.ID) {
		defer log.ExitTestMode()
		log.EnterTestMode()
	}

	for _, account := range user.Accounts {
		if strings.Contains(debugUsers, account.Email) {
			defer log.ExitTestMode()
			log.EnterTestMode()
		}

		logNonNilError(syncAccount(user.ID, account))
	}

	return nil
}

func syncAccount(userID string, account *account) error {
	log.Debug("Started syncing", "userID", userID, "email", account.Email)

	client, err := createCalendarClient(account)
	if err != nil {
		return err
	}

	events, err := client.CalendarEvents(time.Now().UTC().AddDate(0, -1, 0), time.Now().UTC().AddDate(0, 0, 15))
	if err != nil {
		return err
	}

	err = parse.DeleteUserEvents(userID)
	if err != nil {
		return err
	}

	err = parse.PutEvents(userID, events)
	if err != nil {
		return err
	}

	log.Info("Done syncing", "userID", userID, "email", account.Email)
	return nil
}

// createCalendarClient is a calendar client factory function that returns
// the appropriate calendar client for the given user's account.
func createCalendarClient(account *account) (calendar.Client, error) {
	if account.LoginType == "Exchange" {
		loginInfo := strings.Split(account.LoginInfo, " ")
		if len(loginInfo) < 3 {
			return nil, fmt.Errorf("WF00000: Malformed login info: %#v", loginInfo)
		}

		password, err := secrets.RetrieveUserSecret(account.Password)
		if err != nil {
			return nil, err
		}
		return ews.NewClient(loginInfo[2], account.Email, password), nil
	}

	if strings.HasSuffix(account.Host, "outlook.com") || strings.HasSuffix(account.Host, "office365.com") {
		password, err := secrets.RetrieveUserSecret(account.Password)
		if err != nil {
			return nil, err
		}
		return ews.NewClient("https://outlook.office365.com/EWS/Exchange.asmx", account.Email, password), nil
	}

	if account.Host == "imap.gmail.com" {
		return google.NewCalendarClient(account.RefreshToken)
	}

	password, err := secrets.RetrieveUserSecret(account.Password)
	if err != nil {
		return nil, err
	}
	return caldav.NewClient(account.Host, account.Email, password)
}

func waitIndefinitely() {
	// Set up a channel on which to send signal notifications.
	// We must use a buffered channel or risk missing the signal
	// if we're not ready to receive when the signal is sent.
	channel := make(chan os.Signal, 1)
	signal.Notify(channel, os.Interrupt)
	<-channel
}

func logBeforeExiting() {
	if err := recover(); err != nil {
		log.Error("Recovered", "err", err)
	}
	log.Info("Bye!")
}

func logNonNilError(err error) {
	if err != nil {
		log.ErrorObject(err)
	}
}

type user struct {
	ID       string     `json:"objectId"`
	Accounts []*account `json:"imapUsers,omitempty"`
}

type account struct {
	LoginType    string `json:"loginType,omitempty"`
	Host         string `json:"hostname,omitempty"`
	LoginInfo    string `json:"loginId,omitempty"`
	Email        string `json:"email,omitempty"`
	Password     string `json:"password,omitempty"`
	RefreshToken string `json:"refreshToken,omitempty"`
}
