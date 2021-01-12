package blog

import (
	"html/template"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/fbegyn/website/cmd/server/internal/front"
	"github.com/russross/blackfriday"
)

type Entry struct {
	Title      string
	Link       string
	Summary    string
	Body       string
	BodyHTML   template.HTML
	Tags       []string
	Date       time.Time
	DateString string
	Draft      bool
}

type Entries []Entry

// 3 functions needed for sort: Len, Less, Swap
func (e Entries) Len() int { return len(e) }
func (e Entries) Less(i, j int) bool {
	iDate, jDate := e[i].Date, e[j].Date
	return iDate.Unix() < jDate.Unix()
}
func (e Entries) Swap(i, j int) { e[i], e[j] = e[j], e[i] }

func LoadEntriesDir(dirName string, prefix string) (Entries, error) {
	type frontMatter struct {
		Title string
		Date  string
		Tags  []string
		Draft bool
	}

	var entries Entries

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

		content, err := ioutil.ReadAll(file)
		if err != nil {
			return nil
		}

		var fm frontMatter
		post, err := front.Unmarshal(content, &fm)
		if err != nil {
			return err
		}

		postHTML := blackfriday.Run(post)
		const timeFormat = `2006-01-02`
		date, err := time.Parse(timeFormat, fm.Date)
		if err != nil {
			return err
		}

		fileName := filepath.Base(filePath)
		fileName = strings.TrimSuffix(fileName, filepath.Ext(fileName))

		entry := Entry{
			Title:      fm.Title,
			Date:       date,
			DateString: fm.Date,
			Link:       filepath.Join(prefix, fileName),
			Body:       string(post),
			BodyHTML:   template.HTML(postHTML),
			Tags:       fm.Tags,
			Draft:      fm.Draft,
		}

		if !entry.Draft {
		    entries = append(entries, entry)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Sort(sort.Reverse(entries))

	return entries, nil
}
