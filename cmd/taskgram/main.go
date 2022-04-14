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
	"fmt"
	"log"
	"os"
	"time"

	"github.com/nemca/taskgram/internal/config"
	"github.com/nemca/taskgram/internal/helpers"
	"github.com/nemca/taskgram/internal/models"
	"github.com/nemca/taskgram/internal/repository"
)

func main() {
	cfg, err := config.Init()
	if err != nil {
		log.Fatalf("reading config: %v", err)
	}
	if cfg == nil {
		log.Fatalf("config is empty")
	}

	// Check that either only dates or only times
	if len(cfg.Search.LastEditedTimeEnd) > 0 &&
		(len(cfg.Search.LastEditedDateStart) > 0 || len(cfg.Search.LastEditedDateEnd) > 0) {
		fmt.Printf("Please, use either only dates or only times for search.\n")
		os.Exit(1)
	}

	// Prepare times for search requests
	var searchTimeStart, searchTimeEnd time.Time

	// Parse times from config to time.Time
	if len(cfg.Search.LastEditedTimeStart) > 0 {
		lastEditedTimeStart, err := helpers.ParseDuration(cfg.Search.LastEditedTimeStart)
		if err != nil {
			log.Fatalf("convert start last edited time from config: %v", err)
		}
		searchTimeStart = time.Now().Add(-lastEditedTimeStart)
		// If end last edited time not set use now
		searchTimeEnd = time.Now()
		if len(cfg.Search.LastEditedTimeEnd) > 0 {
			lastEditedTimeEnd, err := helpers.ParseDuration(cfg.Search.LastEditedTimeEnd)
			if err != nil {
				log.Fatalf("convert end last edited time from config: %v", err)
			}
			searchTimeEnd = time.Now().Add(-lastEditedTimeEnd)
		}
	}
	// Parse dates from config to time.Time
	if len(cfg.Search.LastEditedDateStart) > 0 {
		lastEditedDateStart, err := helpers.ParseDate(cfg.Search.LastEditedDateStart)
		if err != nil {
			log.Fatalf("convert start last edited date from config: %v", err)
		}
		searchTimeStart = time.Now().Add(-lastEditedDateStart)
		// If end last edited date not set use now
		searchTimeEnd = time.Now()
		if len(cfg.Search.LastEditedDateEnd) > 0 {
			lastEditedDateEnd, err := helpers.ParseDate(cfg.Search.LastEditedDateEnd)
			if err != nil {
				log.Fatalf("convert end last edited date from config: %v", err)
			}
			searchTimeEnd = time.Now().Add(-lastEditedDateEnd)
		}
	}

	// Show the times for search
	fmt.Printf("Finding notes from %q to %q:\n\n", searchTimeStart.Format(time.RFC1123), searchTimeEnd.Format(time.RFC1123))

	var doneTasks models.Tasks
	var todayTasks models.Tasks
	searchConfig := &models.SearchConfig{
		LastEditedTimeStart: searchTimeStart,
		LastEditedTimeEnd:   searchTimeEnd,
	}

	// Targets
	for _, target := range cfg.Targets {
		switch target.Type {
		case "notion":
			notionRepo, err := repository.NewNotionRepository(target.Name, &target.Notion)
			if err != nil {
				log.Fatalf("create Notion repository %s: %v", target.Name, err)
			}

			done, today, err := notionRepo.GetTasks(cfg, searchConfig)
			if err != nil {
				log.Fatalf("get notion tasks from repo: %v", err)
			}
			doneTasks = append(doneTasks, done...)
			todayTasks = append(todayTasks, today...)
		}
	}

	// Print search results
	if doneTasks.NotesLen() > 0 {
		fmt.Println("YESTERDAY:")
		fmt.Println(doneTasks.String())
	}
	if todayTasks.NotesLen() > 0 {
		fmt.Println("TODAY:")
		fmt.Println(todayTasks.String())
	}
}
