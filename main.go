package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
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

func obtainAudio(videoID string) ([]byte, error) {
	audio, err := searchAudioInfo(videoID)
	if err != nil && err == errRecordNotFound {
		// Get Downlodable video URL
		info, infoErr := getVideoInfo(videoID)
		if infoErr != nil {
			return nil, err
		}

		videoURL, downloadURLErr := info.getDownloadableURL()
		if downloadURLErr != nil {
			return nil, err
		}

		// Convert mp4 to mp3 with FFmpeg and save to file
		tmpfile, tmpErr := ioutil.TempFile("", "AudioTmpFile")
		if tmpErr != nil {
			return nil, err
		}
		defer os.Remove(tmpfile.Name())

		if err = downloadAndConvert(videoURL, tmpfile.Name()); err != nil {
			return nil, err
		}

		// Save to Cloud Storage in GCP
		audioPath, putErr := storagePut(videoID, tmpfile)
		if putErr != nil {
			return nil, fmt.Errorf("saving video %s failed, %s", info.title, putErr)
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
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	// Response to a client
	r, exist, err := storageGet(videoID)
	if err != nil {
		return nil, fmt.Errorf("retrieving %s/%s from GCS, %s", bucketName, audio.audioPath, err)
	} else if !exist {
		return nil, fmt.Errorf("%s/%s not found", bucketName, audio.audioPath)
	}
	defer r.Close()
	audioData, err := ioutil.ReadAll(r)
	if err != nil {
		logger.Print(err)
		return nil, err
	}
	return audioData, nil
}

func main() {
	if err := os.MkdirAll(audiosDir, 0777); err != nil && !os.IsExist(err) {
		logger.Fatal(err)
	}

	e := echo.New()
	e.GET("/audio", func(c echo.Context) error {
		videoID := c.QueryParam("id")
		audioData, err := obtainAudio(videoID)
		if err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprint(err))
		}
		return c.Blob(http.StatusOK, "audio/mpeg", audioData)
	})
	e.Logger.Fatal(e.Start(":1234"))
}
