package main

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	_ "github.com/lib/pq"
)

var (
	dbName            string
	userName          string
	password          string
	dbPort            int
	tableName         string
	errRecordNotFound = errors.New("no such record")
)

type audioInfo struct {
	videoID      string
	title        string
	author       string
	thumbnailURL string
	length       int64
	audioURL     string
	keywords     []string
	convertedAt  int64
}

func init() {
	dbName = os.Getenv("DATABASE_NAME")
	userName = os.Getenv("USER_NAME")
	password = os.Getenv("PASSWORD")
	dbPortStr := os.Getenv("DATABASE_PORT")
	tableName = os.Getenv("TABLE_NAME")

	var err error
	dbPort, err = strconv.Atoi(dbPortStr)
	if err != nil {
		logger.Fatal(err)
	}
}

func (info *audioInfo) String() string {
	return fmt.Sprintf("videoID %s\ntitle %s\nauthor%s\nthumbnail from %s\nlength %d sec\nkeywords:%s\nconverted at %d",
		info.videoID, info.title, info.author, info.thumbnailURL, info.length, strings.Join(info.keywords, ", "), info.convertedAt)
}

func (info *audioInfo) encodeKeywords() string {
	var encoded string
	for _, w := range info.keywords {
		encodedWord := strings.Replace(w, ",", "%2C", -1)
		encoded += fmt.Sprintf("%s,", encodedWord)
	}
	return strings.Trim(encoded, ", ")
}

func (info *audioInfo) decodeKeywords(encoded string) {
	encodedWords := strings.Split(encoded, ",")
	words := make([]string, 0, len(encodedWords))
	for _, w := range encodedWords {
		decodedWord := strings.Replace(w, "%2C", ",", -1)
		words = append(words, decodedWord)
	}
	info.keywords = words
}

func openAudioInfoDB() (*sql.DB, error) {
	params := fmt.Sprintf("user=%s password=%s dbname=%s host=localhost port=%d sslmode=disable", userName, password, dbName, dbPort)
	db, err := sql.Open("postgres", params)
	if err != nil {
		logger.Printf("open sql, %s", err)
		return nil, err
	}
	return db, nil
}

func insertAudioInfo(info *audioInfo) error {
	db, err := openAudioInfoDB()
	if err != nil {
		return err
	}
	defer db.Close()

	query := fmt.Sprintf(`INSERT INTO %s(video_id, title, author, thumbnail_url, length, audio_url, keywords, converted_at)
    VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`, tableName)
	stmt, err := db.Prepare(query)
	if err != nil {
		logger.Print(err)
		return err
	}
	_, err = stmt.Exec(info.videoID, info.title, info.author, info.thumbnailURL,
		info.length, info.audioURL, info.encodeKeywords(), info.convertedAt)
	if err != nil {
		logger.Printf("insert audio info into db, %s", err)
	}
	return nil
}

func searchAudioInfo(videoID string) (*audioInfo, error) {
	db, err := openAudioInfoDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	query := fmt.Sprintf(`SELECT * FROM %s WHERE video_id='%s'`, tableName, videoID)
	rows, err := db.Query(query)
	if err != nil {
		logger.Printf("query to search %s, %s", videoID, err)
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		logger.Print(errRecordNotFound)
		return nil, errRecordNotFound
	}
	var (
		info     audioInfo
		keywords string
	)
	if err := rows.Scan(&info.videoID, &info.title, &info.author, &info.thumbnailURL,
		&info.length, &info.audioURL, &keywords, &info.convertedAt); err != nil {
		logger.Printf("parse query result, %s", err)
		return nil, err
	}
	info.decodeKeywords(keywords)
	return &info, nil
}
