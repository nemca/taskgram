# taskgram

Taskgram создан, чтобы упростить ведение заметок о проделанной и запланированной работе в разных местах.
Например вывод `taskgram` в формате Markdown легко скопировать в [Status Hero](https://statushero.com/).

На данный момент реализована поддержка поиска по [Notion](https://www.notion.so) и [Google Calendar](https://calendar.google.com/).
`taskgram` ищет в базе данных Notion, которую вы указали к конфиге, все задачи где вы являетесь исполнителем.
Из этого списка задач отбираются те, которые изменялись в указанный период времени и которые имеют у себя на странице блок заголовок (`Heading`), настраиваемый через переменную конфига `search.headingDoneName`, например `Workflow notes`. Внутри блока отбираются заметки в формате блоков `Text`, `Bullet List` и `Numbered List`. Эти заметки будут в выводе после тега `YESTERDAY:`.
Если в календаре были события, то они добавяться списком с загаловком `Meetings`.

Для тега `TODAY:` отбираются все заметки внутри блока загловка `search.headingToDoName` независимо от времени добавления и все события календаря.

![Notes examples](assets/notes_example.png)

Временной интервал можно задавать через конфигурационный файл или аргементы командной строки. Поддерживаемые форматы даты - `m`, `h`, `d` и `w`.
Например `1w3d10h` - это 1 неделя, 3 дня и 10 часов.

Для доступа в API Notion необходимо получить у администратора токен (подробнее в [документации](https://developers.notion.com/docs/getting-started) Notion). Достаточно иметь права на [чение контента](https://developers.notion.com/reference/capabilities#read-content) из базы и права на получение информации о [пользователях](https://developers.notion.com/reference/capabilities#user-capabilities) без email.

## Build
```
make
```

## Run
```
$ taskgram
Finding notes from "Thu, 31 Mar 2022 21:59:34 +04" to "Fri, 01 Apr 2022 21:59:34 +04":

YESTERDAY:
- [Development taskgram](https://www.notion.so/Development-taskgram-970ce9cf59e94fadbbfd2936d6151bb6)
  - Added block search by name
  - Added bullet list support for notes
  - Added number list support for notes

TODAY:
- [Development taskgram](https://www.notion.so/Development-taskgram-970ce9cf59e94fadbbfd2936d6151bb6)
  - Parsing todo block
- Meetings
  - DevOps daily meetings 2.0 (DDM)
  - Cosmos weekly sync
```

## Config example
`taskgram` searhing config file `.taskgram.yaml` in your home directory.
```yaml
---
targets:
  - name: "ACME board"
    type: "notion"
    notion_config:
      # You API key.
      apiKey: "secret_XXX..."
      # The Database UUID where you store notes.
      databaseID: "E4C05C5C-67E1-46AB-9BB8-7E9FBAD59A4A"
      # Your Notion's user ID.
      # If not set, will try to get ID from Notion by username.
      userID: "26967411-7DD7-49B5-B9F9-437725C91007"
      # Your preferred name in Notion account.
      username: "John Doe"
      # Timeout for Notion's requests.
      timeout: "10s"
      # Name of heading block where you write done notes.
      headingDoneName: "Workflow notes"
      # Name of heading block where you whire todo notes.
      headingToDoName: "TODO"

  - name: "Google calendar"
    type: "google_calendar"
    google_calendar_config:
      calendarID: "johndoe@example.com"
      credentials_path: "/Users/john/.credentials.json"
      token_path: "/Users/john/.google_calendar_token.json"
      timeout: "10s"

search_config:
  # Valid time units are "m", "h", "d", "w"
  # or special words "today" and "yesterday".
  # Dates format is YYYY-MM-DD.
  # Should used either only dates or only times.
  #
  # Start time when notes was last updated.
  lastEditedTimeStart: "today"
  # Start date when notes was last updated.
  lastEditedDateStart: ""
  # End time when notes was last updated.
  # Empty string is mean now.
  lastEditedTimeEnd: "2h"
  # End date when notes was last updated.
  # Empty string is mean now.
  lastEditedDateEnd: ""
```

## Help
```
$ taskgram --help
Usage of ./taskgram:
  -j, --enddate string     End date when notes was last updated.
  -e, --endtime string     End time when notes was last updated.
  -d, --startdate string   Start date when notes was last updated.
  -s, --starttime string   Start time when notes was last updated. (default "24h")
```
