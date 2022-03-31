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

package helpers

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/jomei/notionapi"
)

const (
	propertyFilterAssign string = "Assign"
	propertyDescription  string = "Description"
)

var (
	ErrNotFound = errors.New("not found")
)

// QueryNotionPages returns pages from database which has property Assign equals to user
func QueryNotionPages(client *notionapi.Client, userID, databaseID string) (output []notionapi.Page, err error) {
	var pages []notionapi.Page
	var cursor notionapi.Cursor

	for hasMore := true; hasMore; {
		databaseQueryRequest := &notionapi.DatabaseQueryRequest{
			CompoundFilter: &notionapi.CompoundFilter{
				notionapi.FilterOperatorAND: []notionapi.PropertyFilter{
					{
						Property: propertyFilterAssign,
						People: &notionapi.PeopleFilterCondition{
							Contains: userID,
						},
					},
				},
			},
			StartCursor: cursor,
		}
		resp, err := client.Database.Query(context.Background(), notionapi.DatabaseID(databaseID), databaseQueryRequest)
		if err != nil {
			return nil, errors.New(err.Error())
		}
		pages = append(pages, resp.Results...)
		hasMore = resp.HasMore
		cursor = resp.NextCursor
	}

	return pages, nil
}

// QueryNotionUser find user in Notion by user name.
// Returns ErrNotFound if user not found.
func QueryNotionUser(client *notionapi.Client, username string) (notionapi.User, error) {
	var cursor notionapi.Cursor

	for hasMore := true; hasMore; {
		pagination := &notionapi.Pagination{
			StartCursor: cursor,
			PageSize:    300,
		}
		resp, err := client.User.List(context.Background(), pagination)
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

func QueryNotionSearchHeading(client *notionapi.Client, blockID notionapi.BlockID, searchTime time.Time, name string) (notionapi.Block, error) {
	var cursor notionapi.Cursor

	for hasMore := true; hasMore; {
		pagination := &notionapi.Pagination{
			StartCursor: cursor,
			PageSize:    300,
		}
		resp, err := client.Block.GetChildren(context.Background(), blockID, pagination)
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

func QueryNotionSearchNotes(client *notionapi.Client, blockID notionapi.BlockID, searchTime time.Time) ([]string, error) {
	var notes []string
	var cursor notionapi.Cursor

	for hasMore := true; hasMore; {
		pagination := &notionapi.Pagination{
			StartCursor: cursor,
			PageSize:    300,
		}
		resp, err := client.Block.GetChildren(context.Background(), blockID, pagination)
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

func GetPageTitle(page *notionapi.Page) (string, error) {
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

func NotionQueryGetNotes(client *notionapi.Client, page *notionapi.Page, searchTimeStart time.Time, headingName string, taskCh chan Task, wg *sync.WaitGroup, done chan struct{}) {
	defer wg.Done()
	var task Task
	var err error

	// Title
	task.Title, err = GetPageTitle(page)
	if err != nil {
		log.Printf("get page title: %v", err)
		done <- struct{}{}
		return
	}
	task.URL = page.URL
	// Workflow notes
	// get page content
	pageContent, err := client.Block.Get(context.Background(), notionapi.BlockID(page.ID))
	if err != nil {
		log.Printf("get page content: %v", err)
		done <- struct{}{}
		return
	}
	// Get heading block by name
	heading, err := QueryNotionSearchHeading(client, pageContent.GetID(), searchTimeStart, headingName)
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
	task.Notes, err = QueryNotionSearchNotes(client, heading.GetID(), searchTimeStart)
	if err != nil && !errors.Is(err, ErrNotFound) {
		log.Printf("get workflow notes: %v", err)
		done <- struct{}{}
		return
	}

	taskCh <- task
	done <- struct{}{}
}

func getRichText(rt []notionapi.RichText) (text string) {
	for _, t := range rt {
		text += t.Text.Content
	}
	return
}
