package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const (
	youtubeInfoURL = "https://www.youtube.com/get_video_info?video_id="
	targetType     = "video/mp4"
)

var (
	qualities            = []string{"small", "medium", "hd720"}
	errRedirectAttempted = errors.New("redirect")
)

type stream struct {
	quality   string
	mediaType string
	url       string // video/mp4, video/webcam, etc.
}

type contentInfo struct {
	id             string
	title          string
	author         string
	downloadedTime int64 // Unix time
	thumbnailURL   string
	keywords       []string
	lengthSeconds  int64
	streams        []stream
}

func downloadAndConvert(videoURL, audioPath string) error {
	cmd := exec.Command("ffmpeg", "-i", videoURL, "-acodec", "libmp3lame", "-ab", "256k", "-f", "mp3", audioPath)

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	defer stderr.Close()

	if err = cmd.Start(); err != nil {
		logger.Printf("ffmpeg, %s", err)
		return err
	}

	buf, err := ioutil.ReadAll(stderr)
	if err != nil {
		logger.Printf("connect stderr pipe, %s", err)
		return err
	}
	logger.Print(string(buf[:]))

	if err = cmd.Wait(); err != nil {
		return err
	}

	return nil
}

func getVideoInfo(id string) (*contentInfo, error) {
	infoURL := youtubeInfoURL + id
	res, err := http.Get(infoURL)
	if err != nil {
		logger.Printf("retreive video %s info, %s", id, err)
		return nil, err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		logger.Print(err)
		return nil, err
	}

	parsedInfo, err := url.ParseQuery(string(body))
	if err != nil {
		logger.Print(err)
		return nil, err
	}

	status, ok := parsedInfo["status"]
	if !ok {
		err = fmt.Errorf("no response status found in the server's answer")
		logger.Print(err)
		return nil, err
	}
	if status[0] == "fail" {
		reason, ok := parsedInfo["reason"]
		if ok {
			err = fmt.Errorf("'fail' response status found in the server's answer, reason: '%s'", reason[0])
			logger.Print(err)
		} else {
			logger.Print(err)
			err = errors.New(fmt.Sprint("'fail' response status found in the server's answer, no reason given"))
		}
		return nil, err
	}
	if status[0] != "ok" {
		err = fmt.Errorf("non-success response status found in the server's answer (status: '%s')", status)
		logger.Print(err)
		return nil, err
	}

	lengthSeconds, err := strconv.Atoi(parsedInfo["length_seconds"][0])
	if err != nil {
		lengthSeconds = 0
	}
	streams, err := parseVideoStreams(parsedInfo["url_encoded_fmt_stream_map"][0])
	if err != nil {
		return nil, err
	}

	info := &contentInfo{
		id:             id,
		title:          parsedInfo["title"][0],
		author:         parsedInfo["author"][0],
		downloadedTime: time.Now().Unix(),
		thumbnailURL:   parsedInfo["thumbnail_url"][0],
		keywords:       parsedInfo["keywords"],
		lengthSeconds:  (int64)(lengthSeconds),
		streams:        streams,
	}
	return info, nil
}

func parseVideoStreams(strStreams string) ([]stream, error) {
	// read each stream
	streamsList := strings.Split(strStreams, ",")
	var streams []stream

	for streamPos, streamRaw := range streamsList {
		streamQry, err := url.ParseQuery(streamRaw)
		if err != nil {
			log.Printf("An error occured while decoding one of the video's stream's information: stream %d: %s\n", streamPos, err)
			continue
		}
		streams = append(streams, stream{
			quality:   streamQry["quality"][0],
			mediaType: streamQry["type"][0],
			url:       streamQry["url"][0],
		})
	}
	return streams, nil
}

// getDownloadableURL obtains downloadable url from Youtube
// type video/mp4 is douwnloaded
// seek from lower resolution (small > medium > hd720)
func (info *contentInfo) getDownloadableURL() (string, error) {
	var targetStream *stream

	for _, quality := range qualities {
		for i, s := range info.streams {
			if s.quality == quality && strings.Contains(s.mediaType, targetType) {
				targetStream = &info.streams[i]
				break
			}
		}
	}
	if targetStream == nil {
		err := errors.New("suported stream not found")
		logger.Print(err)
		return "", err
	}

	// Download video
	targetURL := targetStream.url
	redirectionCounter := 0
	reqAttempMax := 10

	for {
		client := &http.Client{
			Timeout: time.Duration(3) * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return errRedirectAttempted
			},
		}
		res, err := client.Head(targetURL)
		if urlErr, ok := err.(*url.Error); ok && urlErr.Err == errRedirectAttempted {
			if redirectionCounter > reqAttempMax {
				err = errors.New("exceeded redirection maximum attemps")
				logger.Print(err)
				return "", err
			}

			loc, locErr := res.Location()
			if locErr != nil {
				logger.Print(err)
				return "", err
			}
			targetURL = loc.RawPath
			redirectionCounter++
			continue
		} else if err != nil {
			logger.Printf("header request, %s", err)
			return "", err
		}

		if res.StatusCode == http.StatusOK {
			break
		} else {
			err := fmt.Errorf("header response status: %s", res.Status)
			logger.Print(err)
			return "", err
		}
	}
	logger.Printf("target url: %s", targetURL)
	return targetURL, nil
}
