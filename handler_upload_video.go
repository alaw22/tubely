package main

import (
	"fmt"
	"os"
	"io"
	"net/http"
	"mime"
	"context"
	"strings"
	"crypto/rand"
	"encoding/base64"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	// Upload limit
	r.Body = http.MaxBytesReader(w, r.Body, 1 << 30)
	defer r.Body.Close()
	
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	// Get video metadata to compare the users_id to the user_id of the video
	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, 400, "Unable to get video", err)
		return 
	}

	if video.CreateVideoParams.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Invalid user", fmt.Errorf("Invalid user"))
		return
	}

	// parse video data
	videoFile, header, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, 500, "Unable to parse video", err)
		return
	}

	defer videoFile.Close()

	mediaType, _, err := mime.ParseMediaType(header.Header.Get("Content-Type"))
	if mediaType != "video/mp4"{
		respondWithError(w, 400, "Video is not in mp4 format", fmt.Errorf("Video is not in mp4 format"))
		return
	}

	// Create unique video file name
	fileExtension := strings.Split(mediaType, "/")[1] // video/mp4
	key := make([]byte,32)
	rand.Read(key)

	encodedKey := make([]byte, base64.RawURLEncoding.EncodedLen(len(key)))
	base64.RawURLEncoding.Encode(encodedKey,key)

	videoName := string(encodedKey) + "." + fileExtension

	tempFile, err := os.CreateTemp("","tubely-video-upload.mp4")
	if err != nil {
		respondWithError(w, 500, "Error creating temporary mp4 file",err)
		return
	}

	defer os.Remove(tempFile.Name())
	defer tempFile.Close() // defer is LIFO so it will close before removing

	
	_, err = io.Copy(tempFile, videoFile)
	if err != nil {
		respondWithError(w, 500, "Error copying contents of video file",err)
		return 
	}

	
	// Rewind temp file pointer
	tempFile.Seek(0, io.SeekStart)
	
	// grab aspect ratio and include it in the url
	aspect, err := getVideoAspectRatio(tempFile.Name())
	if err != nil {
		respondWithError(w, 500, "Error getting aspect ratio from temp file",err)
		return
	}

	var s3Key string
	switch aspect {
	case "16:9":
		s3Key = fmt.Sprintf("landscape/%s",videoName)
	case "9:16":
		s3Key = fmt.Sprintf("portrait/%s",videoName)
	default:
		s3Key = fmt.Sprintf("other/%s",videoName)
	}

	processedFileName, err := processVideoForFastStart(tempFile.Name())
	if err != nil {
		respondWithError(w, 500, "Error encoding video with fast start",err)
		return 
	}
	defer os.Remove(processedFileName)

	processedFile, err := os.Open(processedFileName)
	if err != nil {
		respondWithError(w, 500, "Error opening fast start mp4 file",err)
		return
	}

	defer processedFile.Close()

	// Place video into s3 bucket
	objectInput := s3.PutObjectInput{
		Bucket: &cfg.s3Bucket,
		Key: &s3Key,
		Body: processedFile,
		ContentType: &mediaType,
	}

	_, err = cfg.s3Client.PutObject(context.Background(),&objectInput)
	if err != nil {
		respondWithError(w, 500, "Unable to Put video into s3 bucket", err)
		return
	}

	// videoURL := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s",cfg.s3Bucket,cfg.s3Region,s3Key)
	videoURL := fmt.Sprintf("%s,%s",cfg.s3Bucket,s3Key)

	video.VideoURL = &videoURL
	
	signedVideo, err := cfg.dbVideoToSignedVideo(video)
	if err != nil {
		respondWithError(w, 450, "Unable to sign video URL",err)
		return
	}

	err = cfg.db.UpdateVideo(signedVideo)
	if err != nil {
		respondWithError(w, 500, "Unable to update video URL",err)
		return
	}

	w.WriteHeader(200)

}
