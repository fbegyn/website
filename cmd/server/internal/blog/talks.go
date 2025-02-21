package blog

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/fbegyn/website/cmd/server/internal/front"
)

// Talk representents a workshop or presentation. It's a placeholder with some
// information to put in the list
type Talk struct {
	Title      string
	DateString string
	Slug       string
	Date       time.Time
	Tags       []string
	Draft      bool

	Path    string
	Content string
}

type Talks []Talk

// 3 functions needed for sort: Len, Less, Swap
func (e Talks) Len() int { return len(e) }
func (e Talks) Less(i, j int) bool {
	iDate, jDate := e[i].Date, e[j].Date
	return iDate.Unix() < jDate.Unix()
}
func (e Talks) Swap(i, j int) { e[i], e[j] = e[j], e[i] }

func LoadTalksDir(dirName, prefix string, publishDrafts bool) (Talks, error) {
	var talks Talks
	type frontMatter struct {
		Title string
		Date  string
		Link  string
		Tags  []string
		Draft bool
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
			Title:      fm.Title,
			Date:       date,
			DateString: fm.Date,
			Slug:       strings.TrimSuffix(filePath, filepath.Ext(filePath)),
			Path:       strings.TrimPrefix(filePath, "talks/"),
			Draft:      fm.Draft,
		}
		if !talk.Draft || publishDrafts {
			talks = append(talks, talk)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	sort.Sort(sort.Reverse(talks))

	return talks, nil
}

type TalkFS struct {
	fs.FS
}

func (fsys TalkFS) Open(name string) (fs.File, error) {
	file, err := os.Open(name)
	if err != nil {
		fmt.Println(err)
		return nil, &fs.PathError{Op: "open", Path: name, Err: errors.New("failed to open talksfs")}
	}

	info, _ := file.Stat()
	fmt.Println(info.Size())

	// TODO: clean this up further, but for now this seems to work
	// This can probably be solved by implementing a better io.Seeker but I need to learn about it a bit more
	content, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}
	var temp TalkFile
	talk, err := front.Unmarshal(content, &temp)
	if err != nil {
		return nil, err
	}

	temp.Content = talk
	temp.File = file
	return &temp, nil
}

type TalkFile struct {
	fs.File
	io.Seeker
	io.Reader

	Title      string
	DateString string
	Draft      bool
	Path       string
	Content    []byte
	offset     int64
}

func (file *TalkFile) Seek(offset int64, whence int) (int64, error) {
    switch whence {
    case io.SeekStart:
        file.offset = offset
    case io.SeekCurrent:
        file.offset += offset
    case io.SeekEnd:
        // This requires knowing the file size
        info, err := file.Stat()
        if err != nil {
            return 0, err
        }
        file.offset = info.Size() + offset
    default:
        return 0, fmt.Errorf("invalid whence: %d", whence)
    }

    if file.offset < 0 {
        return 0, fmt.Errorf("negative offset")
    }

    return file.offset, nil
}

func (file *TalkFile) Read(p []byte) (n int, err error) {
	if file.offset >= int64(len(file.Content)) {
		return 0, io.EOF
	}

	n = copy(p, file.Content[file.offset:])
	file.offset += int64(n)
	return n, nil
}

func (file *TalkFile) Close() (err error) { return file.File.Close() }

func (file *TalkFile) Stat() (fs.FileInfo, error) {
	return TalkMeta{
		file:  file.File,
		title: file.Title,
		size:  int64(len(file.Content)),
	}, nil
}

type TalkMeta struct {
	file  fs.File
	title string
	size  int64
}

func (info TalkMeta) Name() string       { return info.title }
func (info TalkMeta) Size() int64        { return info.size }
func (info TalkMeta) Mode() fs.FileMode  { return 0 }
func (info TalkMeta) ModTime() time.Time { return time.Now() }
func (info TalkMeta) IsDir() bool        { return false }
func (info TalkMeta) Sys() any           { return info.file }
