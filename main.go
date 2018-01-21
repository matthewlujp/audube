package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"

	"github.com/labstack/echo"
)

const (
	videosDir = "./videos"
	audiosDir = "./audios"
)

var (
	logger  = log.New(os.Stdout, "[audube]", log.Lshortfile)
	verbose *bool
)

func init() {
	verbose = flag.Bool("v", false, "whether to show detail logs")
	flag.Parse()
}

func convertHandler(c echo.Context) error {
	videoID := c.Param("videoID")

	audioPath := path.Join(audiosDir, fmt.Sprintf("%s.mp3", videoID))
	if !exists(audioPath) {
		videoURL, err := getVideoURL(videoID)
		if err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprint(err))
		}

		if err := convert(videoURL, audioPath); err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprint(err))
		}
	}

	f, err := os.Open(audioPath)
	if err != nil {
		logger.Print(err)
		return c.String(http.StatusInternalServerError, fmt.Sprint(err))
	}
	audioData, err := ioutil.ReadAll(f)
	if err != nil {
		logger.Print(err)
		return c.String(http.StatusInternalServerError, fmt.Sprint(err))
	}
	return c.Blob(http.StatusOK, "audio/mpeg", audioData)
}

func exists(path string) bool {
	if _, err := os.Stat(path); err != nil {
		return false
	}
	return true
}

func main() {
	if err := os.MkdirAll(videosDir, 0777); err != nil && !os.IsExist(err) {
		logger.Fatal(err)
	}
	if err := os.MkdirAll(audiosDir, 0777); err != nil && !os.IsExist(err) {
		logger.Fatal(err)
	}

	e := echo.New()
	e.GET("/:videoID", convertHandler)
	e.Logger.Fatal(e.Start(":1234"))
}
