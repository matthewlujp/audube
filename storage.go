package main

import (
	"context"
	"fmt"
	"io"

	"cloud.google.com/go/storage"
)

const bucketName = "extracted-audios"

func storagePut(videoID string, r io.Reader) (string, error) {
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		logger.Print("new client for storage", err)
		return "", err
	}
	defer client.Close()

	w := client.Bucket(bucketName).Object(videoID).NewWriter(ctx)
	defer w.Close()
	if _, err = io.Copy(w, r); err != nil {
		logger.Printf("write to %s/%s, %s", bucketName, videoID, err)
		return "", err
	}
	return fmt.Sprintf("%s/%s", bucketName, videoID), nil
}

func storageGet(videoID string) (io.ReadCloser, bool, error) {
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		logger.Print("new client for storage", err)
		return nil, false, err
	}
	defer client.Close()

	r, err := client.Bucket(bucketName).Object(videoID).NewReader(ctx)
	if err != nil {
		if err == storage.ErrObjectNotExist {
			logger.Printf("%s/%s not found", bucketName, videoID)
			return nil, false, nil
		}
		logger.Printf("read from %s/%s, %s", bucketName, videoID, err)
		return nil, false, err
	}
	return r, true, nil
}
