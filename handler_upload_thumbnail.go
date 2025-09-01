package main

import (
	"fmt"
	"net/http"
	"io"
	"encoding/base64"

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

	mediaType := header.Header.Get("Content-Type")
	if mediaType == "" {
		respondWithError(w, http.StatusBadRequest, "Content-Type header is empty", err)
		return
	}

	data, err := io.ReadAll(file)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to read content bytes", err)
		return		
	}

	// Encode thumbnail
	base64Data := base64.StdEncoding.EncodeToString(data)

	// Create data URL
	dataURL := fmt.Sprintf("data:%s;base64,%s",mediaType,base64Data)

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

	// Save thumbnail in global map
	// videoThumbnails[videoID] = thumbnail{
	// 	data: data,
	// 	mediaType: mediaType,
	// }

	// thumbnail url
	// thumbnailURL := fmt.Sprintf("http://localhost:%s/api/thumbnails/%s",cfg.port,videoIDString)

	video.ThumbnailURL = &dataURL

	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, 500, "Unable to update video meta data", err)
		return						
	}

	respondWithJSON(w, http.StatusOK, video)

}


/* type Video struct {
	ID           uuid.UUID `json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	ThumbnailURL *string   `json:"thumbnail_url"`
	VideoURL     *string   `json:"video_url"`
	CreateVideoParams
} 
	
type thumbnail struct {
	data      []byte
	mediaType string
}

var videoThumbnails = map[uuid.UUID]thumbnail{}

*/