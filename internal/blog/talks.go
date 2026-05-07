package blog

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/fbegyn/website/internal/front"
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

// LoadTalksDir walks dirName for *.md talk files. The slug is always
// rooted at "talks/" so /talks/{year}/{slug} URLs work regardless of
// where the talks actually live on disk.
func LoadTalksDir(dirName, prefix string, publishDrafts bool) (Talks, error) {
	var talks Talks
	dirName = filepath.Clean(dirName) + string(filepath.Separator)

	type frontMatter struct {
		Title string
		Date  string
		Link  string
		Tags  []string
		Draft bool
	}
	err := filepath.Walk(filepath.Clean(dirName), func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if filepath.Ext(filePath) != ".md" {
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

		relPath := strings.TrimPrefix(filePath, dirName)
		slug := "talks/" + strings.TrimSuffix(relPath, filepath.Ext(relPath))

		talk := Talk{
			Title:      fm.Title,
			Date:       date,
			DateString: fm.Date,
			Slug:       slug,
			Path:       relPath,
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

// TalkFS exposes talk markdown files via an fs.FS. When BaseDir is
// non-empty, an incoming path "talks/<rest>" is rewritten to
// "<BaseDir>/<rest>" before opening; with BaseDir empty the name is
// passed straight to os.Open (the legacy behaviour).
type TalkFS struct {
	fs.FS
	BaseDir string
}

func (fsys TalkFS) Open(name string) (fs.File, error) {
	fullPath := name
	if fsys.BaseDir != "" {
		relativePath := strings.TrimPrefix(name, "talks/")
		fullPath = filepath.Join(fsys.BaseDir, relativePath)
	}

	file, err := os.Open(fullPath)
	if err != nil {
		fmt.Println(err)
		return nil, &fs.PathError{Op: "open", Path: name, Err: errors.New("failed to open talksfs")}
	}

	content, err := io.ReadAll(file)
	if err != nil {
		return nil, &fs.PathError{Op: "open", Path: name, Err: errors.New("failed to open talksfs")}
	}
	var temp TalkFile
	talk, err := front.Unmarshal(content, &temp)
	if err != nil {
		return nil, &fs.PathError{Op: "open", Path: name, Err: errors.New("failed to open talksfs")}
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

type WebsiteResponseWriter struct {
	http.ResponseWriter
	status int
}

func (w *WebsiteResponseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func TalkFSHandler(h http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		temp := &WebsiteResponseWriter{ResponseWriter: w}
		h.ServeHTTP(temp, r)
		if temp.status >= 400 {
			slog.ErrorContext(r.Context(), "failed to open TalkFS and serve talk", "status", temp.status)
		}
	}
}
