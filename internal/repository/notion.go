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
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/jomei/notionapi"
	"github.com/nemca/taskgram/internal/config"
	"github.com/nemca/taskgram/internal/models"
)

const (
	propertyFilterAssign string = "Assign"
	propertyProject      string = "Project"
	propertyDescription  string = "Description"
)

var (
	ErrNotFound = errors.New("not found")
)

type NotionRepository struct {
	Client *notionapi.Client
	Cfg    *config.NotionConfig
	Name   string
}

func NewNotionRepository(name string, cfg *config.NotionConfig) (*NotionRepository, error) {
	client := notionapi.NewClient(notionapi.Token(cfg.APIKey))

	// Search userID by username if it's not set explicitly
	if len(cfg.UserID) < 1 {
		user, err := QueryNotionUser(client, cfg.Username, cfg.Timeout)
		if errors.Is(err, ErrNotFound) {
			return nil, fmt.Errorf("User %q not found in Notion.", cfg.Username)
		}
		if err != nil {
			return nil, err
		}
		cfg.UserID = user.ID.String()
		fmt.Printf("WARNING: You userID in %q Notion target is %q. Please, add this ID to config 'notion_config.userID'.\n\n", name, cfg.UserID)
	}

	return &NotionRepository{
		Client: client,
		Cfg: &config.NotionConfig{
			APIKey:          cfg.APIKey,
			DatabaseID:      cfg.DatabaseID,
			UserID:          cfg.UserID,
			Username:        cfg.Username,
			Timeout:         cfg.Timeout,
			HeadingDoneName: cfg.HeadingDoneName,
			HeadingToDoName: cfg.HeadingToDoName,
		},
		Name: name,
	}, nil
}

func (r *NotionRepository) GetTasks(cfg *config.Config, sc *models.SearchConfig) (doneTasks models.Tasks, todayTasks models.Tasks, err error) {
	wg := new(sync.WaitGroup)
	doneTaskCh := make(chan models.Task)
	todayTaskCh := make(chan models.Task)
	doneCh := make(chan struct{})
	counter := 0

	pageTasks, err := r.GetPages()
	if err != nil {
		return nil, nil, err
	}

	// Get done notes
	for _, notionPage := range pageTasks {
		if notionPage.LastEditedTime.After(sc.LastEditedTimeStart) && notionPage.LastEditedTime.Before(sc.LastEditedTimeEnd) {
			wg.Add(1)
			counter++
			donePage := notionPage
			go r.GetNotes(&donePage, sc.LastEditedTimeStart, r.Cfg.HeadingDoneName, doneTaskCh, wg, doneCh)
		}
		// For todo tasks, we don't need to check last edit time and we will always show them
		wg.Add(1)
		counter++
		todayPage := notionPage
		go r.GetNotes(&todayPage, time.Time{}, r.Cfg.HeadingToDoName, todayTaskCh, wg, doneCh)
	}

	for n := counter; n > 0; {
		select {
		case doneTask := <-doneTaskCh:
			doneTasks = append(doneTasks, doneTask)
		case todayTask := <-todayTaskCh:
			todayTasks = append(todayTasks, todayTask)
		case <-doneCh:
			n--
		}

		if doneTaskCh == nil {
			break
		}
	}

	wg.Wait()

	return doneTasks, todayTasks, nil
}

// GetPages returns pages from database which has property Assign equals to user
func (r *NotionRepository) GetPages() (output []notionapi.Page, err error) {
	var pages []notionapi.Page
	var cursor notionapi.Cursor

	for hasMore := true; hasMore; {
		databaseQueryRequest := &notionapi.DatabaseQueryRequest{
			CompoundFilter: &notionapi.CompoundFilter{
				notionapi.FilterOperatorAND: []notionapi.PropertyFilter{
					{
						Property: propertyFilterAssign,
						People: &notionapi.PeopleFilterCondition{
							Contains: r.Cfg.UserID,
						},
					},
				},
			},
			StartCursor: cursor,
		}

		ctx, cancel := context.WithTimeout(context.Background(), r.Cfg.Timeout)
		defer cancel()

		resp, err := r.Client.Database.Query(ctx, notionapi.DatabaseID(r.Cfg.DatabaseID), databaseQueryRequest)
		if err != nil {
			return nil, errors.New(err.Error())
		}

		pages = append(pages, resp.Results...)
		hasMore = resp.HasMore
		cursor = resp.NextCursor
	}

	return pages, nil
}

// GetNotes
func (r *NotionRepository) GetNotes(page *notionapi.Page, searchTimeStart time.Time, headingName string, taskCh chan models.Task, wg *sync.WaitGroup, done chan struct{}) {
	defer wg.Done()

	var task models.Task
	var err error

	// Title
	task.Title, err = getPageTitle(page)
	if err != nil {
		log.Printf("get page title: %v", err)
		done <- struct{}{}
		return
	}
	task.URL = page.URL
	// Try to get page property Project with type multiselect
	// and read projects to task.Projects for tags
	if projectProperty, ok := page.Properties[propertyProject]; ok {
		// cast to MiltiSelectProperty interface
		if project, ok := projectProperty.(*notionapi.MultiSelectProperty); ok {
			for _, ms := range project.MultiSelect {
				task.Projects = append(task.Projects, ms.Name)
			}
		}
	}
	// Workflow notes
	// get page content
	ctx, cancel := context.WithTimeout(context.Background(), r.Cfg.Timeout)
	defer cancel()

	pageContent, err := r.Client.Block.Get(ctx, notionapi.BlockID(page.ID))
	if err != nil {
		log.Printf("get page content: %v", err)
		done <- struct{}{}
		return
	}
	// Get heading block by name
	heading, err := r.SearchHeading(pageContent.GetID(), searchTimeStart, headingName)
	if errors.Is(err, ErrNotFound) {
		done <- struct{}{}
		return
	}
	if err != nil && !errors.Is(err, ErrNotFound) {
		fmt.Printf("get children headings: %v", err)
		done <- struct{}{}
		return
	}
	// Get notes
	task.Notes, err = r.SearchNotes(heading.GetID(), searchTimeStart)
	if err != nil && !errors.Is(err, ErrNotFound) {
		log.Printf("get workflow notes: %v", err)
		done <- struct{}{}
		return
	}

	taskCh <- task
	done <- struct{}{}
}

func (r *NotionRepository) SearchHeading(blockID notionapi.BlockID, searchTime time.Time, name string) (notionapi.Block, error) {
	var cursor notionapi.Cursor

	for hasMore := true; hasMore; {
		pagination := &notionapi.Pagination{
			StartCursor: cursor,
			PageSize:    300,
		}

		ctx, cancel := context.WithTimeout(context.Background(), r.Cfg.Timeout)
		defer cancel()

		resp, err := r.Client.Block.GetChildren(ctx, blockID, pagination)
		if err != nil {
			return nil, err
		}
		for _, block := range resp.Results {
			switch block.GetType() {
			case "heading_1":
				if h, ok := block.(*notionapi.Heading1Block); ok {
					if block.GetLastEditedTime().After(searchTime) {
						if getRichText(h.Heading1.Text) == name {
							return h, nil
						}
					}
				}
			case "heading_2":
				if h, ok := block.(*notionapi.Heading2Block); ok {
					if block.GetLastEditedTime().After(searchTime) {
						if getRichText(h.Heading2.Text) == name {
							return h, nil
						}
					}
				}
			case "heading_3":
				if h, ok := block.(*notionapi.Heading3Block); ok {
					if block.GetLastEditedTime().After(searchTime) {
						if getRichText(h.Heading3.Text) == name {
							return h, nil
						}
					}
				}
			}
		}
		hasMore = resp.HasMore
		cursor = notionapi.Cursor(resp.NextCursor)
	}
	// not found
	return nil, ErrNotFound
}

func (r *NotionRepository) SearchNotes(blockID notionapi.BlockID, searchTime time.Time) ([]string, error) {
	var notes []string
	var cursor notionapi.Cursor

	for hasMore := true; hasMore; {
		pagination := &notionapi.Pagination{
			StartCursor: cursor,
			PageSize:    300,
		}

		ctx, cancel := context.WithTimeout(context.Background(), r.Cfg.Timeout)
		defer cancel()

		resp, err := r.Client.Block.GetChildren(ctx, blockID, pagination)
		if err != nil {
			return nil, err
		}
		for _, block := range resp.Results {
			switch block.GetType() {
			case "paragraph":
				if paragraphBlock, ok := block.(*notionapi.ParagraphBlock); ok {
					if paragraphBlock.GetLastEditedTime().After(searchTime) {
						notes = append(notes, getRichText(paragraphBlock.Paragraph.Text))
					}
				}
			case "bulleted_list_item":
				if bulletListItem, ok := block.(*notionapi.BulletedListItemBlock); ok {
					if bulletListItem.GetLastEditedTime().After(searchTime) {
						notes = append(notes, getRichText(bulletListItem.BulletedListItem.Text))
					}
				}
			case "numbered_list_item":
				if numberListItem, ok := block.(*notionapi.NumberedListItemBlock); ok {
					if numberListItem.GetLastEditedTime().After(searchTime) {
						notes = append(notes, getRichText(numberListItem.NumberedListItem.Text))
					}
				}
			}
		}
		hasMore = resp.HasMore
		cursor = notionapi.Cursor(resp.NextCursor)
	}
	// Not found
	return notes, ErrNotFound
}

// QueryNotionUser find user in Notion by user name.
// Returns ErrNotFound if user not found.
func QueryNotionUser(client *notionapi.Client, username string, timeout time.Duration) (notionapi.User, error) {
	var cursor notionapi.Cursor

	for hasMore := true; hasMore; {
		pagination := &notionapi.Pagination{
			StartCursor: cursor,
			PageSize:    300,
		}

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		resp, err := client.User.List(ctx, pagination)
		if err != nil {
			return notionapi.User{}, err
		}
		for _, u := range resp.Results {
			// found user and return
			if u.Name == username {
				return u, nil
			}
		}
		hasMore = resp.HasMore
		cursor = resp.NextCursor
	}

	// Not found
	return notionapi.User{}, ErrNotFound
}
func getPageTitle(page *notionapi.Page) (string, error) {
	if page == nil {
		return "", fmt.Errorf("cannot read title, nil page")
	}

	descriptionProperty := page.Properties[propertyDescription]
	if descriptionProperty == nil {
		return "", fmt.Errorf("cannot read description, invalid property")
	}
	// cast to TitleProperty interface
	p, ok := descriptionProperty.(*notionapi.TitleProperty)
	if !ok {
		return "", fmt.Errorf("cannot read title, not a page title property")
	}
	if len(p.Title) < 1 {
		// On page without any title, the internal struct is empty.
		return "", nil
	}

	var title string
	for _, t := range p.Title {
		title += t.PlainText
	}

	return title, nil
}

func getRichText(rt []notionapi.RichText) (text string) {
	for _, t := range rt {
		text += t.Text.Content
	}
	return
}
