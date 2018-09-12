package ephi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/lestrrat-go/slack"
	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"

	"google.golang.org/appengine/taskqueue"
	"google.golang.org/appengine/urlfetch"

	"google.golang.org/appengine/log"
)

type AppConfig struct {
	Token              string
	AccessToken        string
	DefaultDelay       int
}

type slashResponse struct {
	ResponseType string `json:"response_type"`
	Text         string `json:"text"`
}

type Event struct {
	ID         string          `json:"id"`
	Created    int64           `json:"created"`
	ObjectType string          `json:"objectType"`
	EventType  string          `json:"type"`
	Data       json.RawMessage `json:"data"`
	Log        string          `json:"log"`
}


func init() {

	http.HandleFunc("/slack/ephe", handleEphi)
	http.HandleFunc("/slack/del", handleDeleteMsg)
	http.HandleFunc("/_ah/warmup", handleWarmup)
}



func handleEphi(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)
	if Token != "" && r.PostFormValue("token") != Token {
		http.Error(w, "Invalid Slack token.", http.StatusBadRequest)
		return
	}

	var msg string
	w.Header().Set("content-type", "application/json")
	result := strings.Split(r.PostFormValue("text"), " ")
	msg = r.PostFormValue("text")
	var delay int
	var err error
	delay = DefaultDelay
	if strings.HasPrefix(result[0], "+") {
		delay, err = strconv.Atoi(result[0][1:])
		if err != nil {
			delay = DefaultDelay
		} else {
			msg = strings.Join(result[1:], " ")
		}
	}

	httpCl := urlfetch.Client(ctx)
	slackCl := slack.New(AccessToken, slack.WithClient(httpCl))
	chatres, err := slackCl.Chat().PostMessage(r.PostFormValue("channel_id")).Text(msg).AsUser(true).Do(ctx)
	if err != nil {
		log.Infof(ctx, "failed to post messsage: %s\n", err)
		return
	}
	postValues := url.Values{}
	m := chatres.Message.(map[string]interface{})["ts"]
	postValues.Set("timestamp", m.(string))
	postValues.Set("channel_id", chatres.Channel)
	postValues.Set("token", AccessToken)
	msg = fmt.Sprintf("Your message will be deleted in %d seconds", delay)
	t := taskqueue.NewPOSTTask("/slack/del", postValues)
	t.Delay = time.Duration(delay) * time.Second
	taskqueue.Add(ctx, t, "")


	resp := &slashResponse{
		Text: msg,
	}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Errorf(ctx, "Error encoding JSON: %s", err)
		http.Error(w, "Error encoding JSON.", http.StatusInternalServerError)
		return
	}

}

func handleDeleteMsg(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)
	httpCl := urlfetch.Client(ctx)
	slackCl := slack.New(r.FormValue("token"), slack.WithClient(httpCl))
	_, err := slackCl.Chat().Delete(r.FormValue("channel_id")).Timestamp(r.FormValue("timestamp")).AsUser(true).Do(ctx)
	if err != nil {
		log.Errorf(ctx, err.Error())
	}

}

func handleWarmup(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)
	// Get config data from db.
	key := datastore.NewKey(ctx, "appconfig", "onetime", 0, nil)
	var appconfig AppConfig
	err := datastore.Get(ctx, key, &appconfig)
	if err != nil {
		log.Errorf(ctx, err.Error())
	}
	SetupConfig(ctx, appconfig)
	log.Infof(ctx, "warmup done")

}

