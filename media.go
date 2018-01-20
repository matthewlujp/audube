package main

import (
	"fmt"
	"io/ioutil"
	"os/exec"

	"github.com/kkdai/youtube"
)

func downloadVideo(videoID, filePath string) error {
	logger.Printf("download to file: %s", filePath)

	y := youtube.NewYoutube(*verbose)
	if err := y.DecodeURL(fmt.Sprintf("https://www.youtube.com/watch?v=%s", videoID)); err != nil {
		logger.Printf("decode url %s", err)
		return err
	}
	if err := y.StartDownload(filePath); err != nil {
		logger.Printf("download, %s", err)
		return err
	}
	return nil
}

func convert(videoPath, audioPath string) error {
	cmd := exec.Command("ffmpeg", "-i", videoPath, "-acodec", "libmp3lame", "-ab", "256k", audioPath)

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
