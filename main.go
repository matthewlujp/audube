package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"time"

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

func allocateFile(videoID string) string {
	return path.Join(audiosDir, fmt.Sprintf("%s.mp3", videoID))
}

func openAudio(audioURL string) (io.Reader, error) {
	return os.Open(audioURL)
}

func convertHandler(c echo.Context) error {
	videoID := c.Param("videoID")

	audio, err := searchAudioInfo(videoID)
	if err != nil && err == errRecordNotFound {
		// Downlod video from Youtube and convert it to mp3
		audioPath := allocateFile(videoID)
		info, infoErr := getVideoInfo(videoID)
		if infoErr != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprint(err))
		}

		videoURL, downloadURLErr := info.getDownloadableURL()
		if downloadURLErr != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprint(err))
		}

		if err = downloadAndConvert(videoURL, audioPath); err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprint(err))
		}

		audio = &audioInfo{
			videoID:      videoID,
			title:        info.title,
			author:       info.author,
			thumbnailURL: info.thumbnailURL,
			length:       info.lengthSeconds,
			audioURL:     audioPath,
			keywords:     info.keywords,
			convertedAt:  time.Now().Unix(),
		}
		logger.Print(audio)
		if err = insertAudioInfo(audio); err != nil {
			logger.Print(err)
		}
	} else if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprint(err))
	}

	// Response to a client
	r, err := openAudio(audio.audioURL)
	if err != nil {
		logger.Print(err)
		return c.String(http.StatusInternalServerError, fmt.Sprint(err))
	}
	audioData, err := ioutil.ReadAll(r)
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
