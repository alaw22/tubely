package main

import (
	"io"
	"os"
	"fmt"
	"strings"
	"net/http"
	"path/filepath"
	"mime"
	"encoding/base64"
	"crypto/rand"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
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


	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	// TODO: implement the upload here
	const maxMemory = 10 << 20
	r.ParseMultipartForm(maxMemory)

	// "thumbnail" should match the HTML form input name
	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return
	}
	defer file.Close()

	mediaType, _, err := mime.ParseMediaType(header.Header.Get("Content-Type"))

	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to obtain media type", err)
		return
	}

	if mediaType != "image/jpeg" && mediaType != "image/png"{
		respondWithError(w,
						 http.StatusBadRequest,
						 "Expecting image/jpeg or image/png",
						 fmt.Errorf("Expecting image/jpeg or image/png but got %s",
						 			mediaType))
		return 
	}

	// Get file extension and create a filepath in assests directory
	fileExtension := strings.Split(mediaType,"/")[1] // image/png
	key := make([]byte,32)
	rand.Read(key)

	encodedKey := make([]byte, base64.RawURLEncoding.EncodedLen(len(key)))
	base64.RawURLEncoding.Encode(encodedKey,key)

	filePath := filepath.Join(cfg.assetsRoot,string(encodedKey) + "." + fileExtension) // assets/6dccdc00-f8ab-4bda-bc03-39a27f926558.png
	

	// Create file on disk
	outputFile, err := os.Create(filePath)
	if err != nil {
		respondWithError(w, 500, "Unable to create byte file", err)
		return				
	}

	defer outputFile.Close()

	// Copy file contents
	_, err = io.Copy(outputFile, file)
	if err != nil {
		respondWithError(w, 500, "Unable to copy contents of file", err)
		return					
	}

	// Get video meta-data from video ID
	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, 500, "Video doesn't exist", err)
		return				
	}

	if video.CreateVideoParams.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "User is not the owner of video", err)
		return				
	}

	dataURL := fmt.Sprintf("http://localhost:%s/%s",cfg.port,filePath)
	video.ThumbnailURL = &dataURL

	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, 500, "Unable to update video meta data", err)
		return						
	}

	respondWithJSON(w, http.StatusOK, video)

}
