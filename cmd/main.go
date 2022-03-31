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

package main

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/jomei/notionapi"
	"github.com/nemca/taskgram/internal/config"
	"github.com/nemca/taskgram/internal/helpers"
)

func main() {
	cfg, err := config.Init()
	if err != nil {
		log.Fatalf("reading config: %v", err)
	}
	if cfg == nil {
		log.Fatalf("config is empty")
	}

	// Prepare times for search requests
	var searchTimeStart, searchTimeEnd time.Time
	lastEditedTimeStart, err := helpers.ParseDuration(cfg.Search.LastEditedTimeStart)
	if err != nil {
		log.Fatalf("convert start last edited time from config: %v", err)
	}
	searchTimeStart = time.Now().Add(-lastEditedTimeStart)
	// If end last edited time not set use now
	searchTimeEnd = time.Now()
	if len(cfg.Search.LastEditedTimeEnd) > 1 {
		lastEditedTimeEnd, err := helpers.ParseDuration(cfg.Search.LastEditedTimeEnd)
		if err != nil {
			log.Fatalf("convert end last edited time from config: %v", err)
		}
		searchTimeEnd = time.Now().Add(-lastEditedTimeEnd)
	}

	// Notion client
	client := notionapi.NewClient(notionapi.Token(cfg.Notion.APIKey))

	// Search userID by username if it's not set explicitly
	// and stop if user not found
	userid := cfg.Notion.UserID
	if len(userid) < 1 {
		user, err := helpers.QueryNotionUser(client, cfg.Notion.Username)
		if errors.Is(err, helpers.ErrNotFound) {
			fmt.Printf("User %q not found in Notion.\nPlease, check config file 'notion.username' settings.\n", cfg.Notion.Username)
			return
		}
		if err != nil {
			log.Fatalf("get user: %v", err)
		}
		userid = user.ID.String()
		fmt.Printf("WARNING: You userID is %q. Please, add this ID to config 'nition.userID'.\n\n", userid)
	}

	// Show the times for search
	fmt.Printf("Finding notes from %q to %q:\n\n", searchTimeStart.Format(time.RFC1123), searchTimeEnd.Format(time.RFC1123))
	// and start search
	tasks, err := helpers.QueryNotionPages(client, userid, cfg.Notion.DatabaseID)
	if err != nil {
		log.Fatalf("get tasks: %v", err)
	}

	todayCh := make(chan helpers.Task)
	doneCh := make(chan helpers.Task)
	done := make(chan struct{})
	wg := new(sync.WaitGroup)
	counter := 0

	// Get done notes
	for _, notionPage := range tasks {
		if notionPage.LastEditedTime.After(searchTimeStart) && notionPage.LastEditedTime.Before(searchTimeEnd) {
			wg.Add(1)
			counter++
			c := client
			page := notionPage
			go helpers.NotionQueryGetNotes(c, &page, searchTimeStart, cfg.Search.HeadingDoneName, doneCh, wg, done)
		}
	}

	// Get todo notes
	for _, notionPage := range tasks {
		// For todo tasks, we don't need to check last edit time and we will always show them
		wg.Add(1)
		counter++
		c := client
		page := notionPage
		go helpers.NotionQueryGetNotes(c, &page, time.Time{}, cfg.Search.HeadingToDoName, todayCh, wg, done)
	}

	var doneTasks helpers.Tasks
	var todayTasks helpers.Tasks
	for n := counter; n > 0; {
		select {
		case doneTask := <-doneCh:
			doneTasks = append(doneTasks, doneTask)
		case todayTask := <-todayCh:
			todayTasks = append(todayTasks, todayTask)
		case <-done:
			n--
		}

		if todayCh == nil {
			break
		}
	}
	wg.Wait()

	// Print search results
	if len(doneTasks) > 0 {
		fmt.Println("YESTERDAY:")
		fmt.Println(doneTasks.String())
	}
	if len(todayTasks) > 0 {
		fmt.Println("TODAY:")
		fmt.Println(todayTasks.String())
	}
}
