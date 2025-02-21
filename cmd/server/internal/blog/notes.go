package blog

import (
	"html/template"
	"os"
	"io"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/fbegyn/website/cmd/server/internal/front"
	"github.com/russross/blackfriday"
)

type Note struct {
	Title      string
	Link       string
	Body       string
	BodyHTML   template.HTML
	Tags       []string
	Date       time.Time
	DateString string
	Draft      bool
}

type Notes []Note

// 3 functions needed for sort: Len, Less, Swap
func (e Notes) Len() int { return len(e) }
func (e Notes) Less(i, j int) bool {
	iDate, jDate := e[i].Date, e[j].Date
	return iDate.Unix() < jDate.Unix()
}
func (e Notes) Swap(i, j int) { e[i], e[j] = e[j], e[i] }

func LoadNotesDir(dirName string, prefix string, publishDrafts bool) (Notes, error) {
	type frontMatter struct {
		Title string
		Date  string
		Tags  []string
		Draft bool
	}

	var notes Notes

	err := filepath.Walk(dirName, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		file, err := os.Open(filePath)
		if err != nil {
			return err
		}
		defer file.Close()

		content, err := io.ReadAll(file)
		if err != nil {
			return nil
		}

		var fm frontMatter
		post, err := front.Unmarshal(content, &fm)
		if err != nil {
			return err
		}

		noteHTML := blackfriday.Run(post)
		const timeFormat = `2006-01-02`
		date, err := time.Parse(timeFormat, fm.Date)
		if err != nil {
			return err
		}

		fileName := filepath.Base(filePath)
		fileName = strings.TrimSuffix(fileName, filepath.Ext(fileName))

		note := Note{
			Title:      fm.Title,
			Date:       date,
			DateString: fm.Date,
			Link:       filepath.Join(prefix, fileName),
			Body:       string(post),
			BodyHTML:   template.HTML(noteHTML),
			Tags:       fm.Tags,
			Draft:      fm.Draft,
		}

		if !note.Draft || publishDrafts {
		    notes = append(notes, note)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Sort(sort.Reverse(notes))

	return notes, nil
}
