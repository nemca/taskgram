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
