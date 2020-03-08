package urldownloader

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/google/uuid"
)

// Options

type Options struct {
	maxSize    int64
	baseFolder string
	mimeType   string
	mimeGroups []string
}

func (o *Options) SetMaxSize(size int64) {
	o.maxSize = size
}

func (o *Options) SetBaseFolder(folder string) {
	o.baseFolder = folder
}

func (o *Options) SetMimeType(mime string) {
	o.mimeType = mime
}

func (o *Options) SetMimeGroups(groups ...string) {
	o.mimeGroups = groups
}

var ErrMaxSizeExceeded error = errors.New("Max byte size exceeded")
var ErrUnknownFilename error = errors.New("Unknown filename")
var ErrWrongMimeType error = errors.New("Wrong mime type")
var ErrWrongMimeGroup error = errors.New("Wrong mime group")

func DownloadFileFromUrl(url *url.URL, options ...func(*Options) error) (string, error) {

	var id string
	var path string
	var filename string
	var fullpath string
	var err error
	var pathErr error
	var defaultOptions *Options
	var op func(*Options) error
	var mimeType string

	// options

	defaultOptions = &Options{
		0,
		"/tmp",
		"",
		nil,
	}

	for _, op = range options {
		if err = op(defaultOptions); err != nil {
			return "", err
		}
	}

	// id

	filename = getFilenameFromUrl(url)

	id = uuid.New().String()
	path = fmt.Sprintf("%v/%v", defaultOptions.baseFolder, id)
	if err = os.MkdirAll(path, 0755); err != nil {
		return "", err
	}

	fullpath = fmt.Sprintf("%v/%v", path, filename)
	if err = downloadFile(fullpath, url, defaultOptions.maxSize); err != nil {
		if pathErr = os.RemoveAll(path); err != nil {
			log.Println(pathErr)
		}
		return "", err
	}

	if defaultOptions.mimeType != "" || defaultOptions.mimeGroups != nil {
		if mimeType, err = getMimeType(fullpath); err != nil {
			return "", err
		}

		if defaultOptions.mimeType != "" && defaultOptions.mimeType != mimeType {
			return "", ErrWrongMimeType
		} else if !containsMimeGroup(mimeType, defaultOptions.mimeGroups) {
			return "", ErrWrongMimeGroup
		}
	}

	return fullpath, nil

}

func containsMimeGroup(typ string, group []string) bool {

	var item string
	var result string

	for _, item = range group {
		result = fmt.Sprintf("%v/", item)
		if result == typ[:len(result)] {
			return true
		}
	}

	return false

}

func getMimeType(fullpath string) (string, error) {

	var file *os.File
	var err error
	var buf []byte
	var amount int
	var result string

	if file, err = os.Open(fullpath); err != nil {
		return "", err
	}

	buf = make([]byte, 512)
	if amount, err = file.Read(buf); err != nil {
		return "", err
	}

	result = http.DetectContentType(buf[:amount])

	return result, nil

}

func downloadFile(filename string, url *url.URL, maxSize int64) error {

	var file *os.File
	var response *http.Response
	var err error

	if file, err = os.Create(filename); err != nil {
		return err
	}

	defer file.Close()

	if response, err = http.Get(url.String()); err != nil {
		return err
	}

	defer response.Body.Close()

	if maxSize == 0 {
		if _, err = io.Copy(file, response.Body); err != nil {
			return err
		}
	} else {
		if err = copyMax(file, response.Body, maxSize); err != nil {
			return err
		}
	}

	return nil

}

func copyMax(dst io.Writer, src io.Reader, n int64) error {

	var err error
	var nextByte []byte
	var nRead int

	if _, err = io.CopyN(dst, src, n); err != nil {
		return err
	}

	nextByte = make([]byte, 1)
	nRead, _ = io.ReadFull(src, nextByte)

	if nRead > 0 {
		return ErrMaxSizeExceeded
	}

	return nil

}

func getFilenameFromUrl(url *url.URL) string {

	var result string

	result = strings.Trim(path.Base(url.Path))

	if result == "" {
		return "index.htm"
	} else {
		return result
	}

}
