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

package models

import (
	"bytes"
	"fmt"
	"strings"
)

// Task represents a task
type Task struct {
	Title    string
	URL      string
	Projects []string
	Notes    []string
}

// Tasks represents list of tasks
type Tasks []Task

// String implements fmt.Stringer interface
func (t *Task) String() string {
	var buf bytes.Buffer

	fmt.Fprintf(&buf, "- [%s](%s) ", t.Title, t.URL)
	for _, p := range t.Projects {
		fmt.Fprintf(&buf, "#%s ", strings.ToLower(p))
	}
	fmt.Fprintln(&buf)

	for _, note := range t.Notes {
		fmt.Fprintf(&buf, "  - %s\n", note)
	}

	return buf.String()
}

// String implements fmt.Stringer interface
func (t *Tasks) String() string {
	var buf bytes.Buffer
	for _, task := range *t {
		if len(task.Notes) > 0 {
			fmt.Fprint(&buf, task.String())
		}
	}

	return buf.String()
}

// NotesLen returns number of notes in tasks list
func (t *Tasks) NotesLen() (counter int) {
	for _, task := range *t {
		counter += len(task.Notes)
	}
	return
}
