/*
Copyright © 2022 Michael Bruskov <mixanemca@yandex.ru>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/nemca/taskgram/internal/config"
	"github.com/nemca/taskgram/internal/models"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

type CalendarRepository struct {
	Name    string
	Service *calendar.Service
	Cfg     *config.GoogleCalendarConfig
}

func NewCalendarRepository(name string, cfg *config.GoogleCalendarConfig) (*CalendarRepository, error) {
	ctx := context.Background()

	b, err := ioutil.ReadFile(cfg.CredentialsPath)
	if err != nil {
		return nil, fmt.Errorf("Unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(b, calendar.CalendarReadonlyScope)
	if err != nil {
		return nil, fmt.Errorf("Unable to parse client secret file to config: %v", err)
	}

	client := getClient(config, cfg.TokenPath)

	srv, err := calendar.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("Unable to retrieve Calendar client: %v", err)
	}

	return &CalendarRepository{
		Name:    name,
		Service: srv,
		Cfg:     cfg,
	}, nil
}

func (r *CalendarRepository) GetEvents(cfg *config.Config, sc *models.SearchConfig) (doneEvents models.Events, todayEvents models.Events, err error) {
	timeMin := sc.LastEditedTimeStart.Format(time.RFC3339)
	timeMax := sc.LastEditedTimeEnd.Format(time.RFC3339)

	events, err := r.Service.Events.List("mikhail.b@p2p.org").ShowDeleted(false).
		SingleEvents(true).TimeMin(timeMin).TimeMax(timeMax).MaxResults(10).OrderBy("startTime").Do()
	if err != nil {
		return nil, nil, fmt.Errorf("Unable to retrieve next ten of the user's events: %v", err)
	}

	if len(events.Items) > 0 {
		for _, item := range events.Items {
			date, _ := time.Parse(time.RFC3339, item.Start.DateTime)
			event := models.Event{
				Summary: item.Summary,
			}
			if date.Before(time.Now()) {
				doneEvents = append(doneEvents, event)
			} else {
				todayEvents = append(todayEvents, event)
			}
		}
	}

	return
}

// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config, tokenPath string) *http.Client {
	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	// tokFile := "token.json"
	tok, err := tokenFromFile(tokenPath)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokenPath, tok)
	}
	return config.Client(context.Background(), tok)
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code: %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}
	return tok
}

// Retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	_ = json.NewEncoder(f).Encode(token)
}
