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
	audiosDir = "./audios"
	projectID = "audiube"
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
		// Get Downlodable video URL
		info, infoErr := getVideoInfo(videoID)
		if infoErr != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprint(err))
		}

		videoURL, downloadURLErr := info.getDownloadableURL()
		if downloadURLErr != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprint(err))
		}

		// Convert mp4 to mp3 with FFmpeg and save to file
		tmpfile, tmpErr := ioutil.TempFile("", "AudioTmpFile")
		if tmpErr != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprint(err))
		}
		defer tmpfile.Close()

		if err = downloadAndConvert(videoURL, tmpfile.Name()); err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprint(err))
		}

		// Save to Cloud Storage in GCP
		audioPath, putErr := storagePut(videoID, tmpfile)
		if putErr != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("saving video %s failed, %s", info.title, err))
		}

		audio = &audioInfo{
			videoID:      videoID,
			title:        info.title,
			author:       info.author,
			thumbnailURL: info.thumbnailURL,
			length:       info.lengthSeconds,
			audioPath:    audioPath,
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
	r, exist, err := storageGet(videoID)
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprint(err))
	} else if !exist {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("%s not exist", audio.audioPath))
	}
	defer r.Close()
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
	if err := os.MkdirAll(audiosDir, 0777); err != nil && !os.IsExist(err) {
		logger.Fatal(err)
	}

	e := echo.New()
	e.GET("/:videoID", convertHandler)
	e.Logger.Fatal(e.Start(":1234"))
}
