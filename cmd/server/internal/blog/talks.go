package blog

import (
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/fbegyn/website/cmd/server/internal/front"
)

// Talk representents a workshop or presentation. It's a placeholder with some
// information to put in the list
type Talk struct {
	Title      string
	DateString string
	Date       time.Time
	Tags       []string
	Link       string
}

type Talks []Talk

// 3 functions needed for sort: Len, Less, Swap
func (e Talks) Len() int { return len(e) }
func (e Talks) Less(i, j int) bool {
	iDate, jDate := e[i].Date, e[j].Date
	return iDate.Unix() < jDate.Unix()
}
func (e Talks) Swap(i, j int) { e[i], e[j] = e[j], e[i] }

func LoadTalksDir(dirName, prefix string) (Talks, error) {
	var talks Talks
	type frontMatter struct {
		Title string
		Date  string
		Link  string
		Tags  []string
	}
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
		_, err = front.Unmarshal(content, &fm)
		if err != nil {
			return err
		}

		const timeFormat = `2006-01-02`
		date, err := time.Parse(timeFormat, fm.Date)
		if err != nil {
			return err
		}

		talk := Talk{
			Title: fm.Title,
			Date: date,
			DateString: fm.Date,
			Link: fm.Link,
			Tags: fm.Tags,
		}
		talks = append(talks, talk)
		return nil
	})

	if err != nil {
		return nil, err
	}

	sort.Sort(sort.Reverse(talks))

	return talks, nil
}
