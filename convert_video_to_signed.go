package main

import (
	"fmt"
	"time"
	"strings"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/database"

)

func (cfg *apiConfig) dbVideoToSignedVideo(video database.Video) (database.Video, error) {

	if video.VideoURL == nil{
		return video, nil
	}

	splitURL := strings.Split(*video.VideoURL,",")

	if len(splitURL) < 2 {
		return video, nil
	}

	bucket := splitURL[0]
	key := splitURL[1]

	fmt.Printf("DEBUG: bucket='%s' (length: %d)\n", bucket, len(bucket))
	fmt.Printf("DEBUG: key='%s'\n", key)

	signedURL, err := generatePresignedURL(cfg.s3Client, bucket, key, time.Minute * 5 )

	if err != nil {
		return database.Video{}, err
	}

	video.VideoURL = &signedURL

	return video, nil

}