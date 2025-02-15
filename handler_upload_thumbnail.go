package main

import (
	"fmt"
	"net/http"
	"io"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
	"strings"
	"path/filepath"
	"os"
	"crypto/rand"
	"encoding/base64"
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
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Failed trying to Upload Thumbnail", err)
		return
	}
	videoData, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Failed to fetch Metadata", err)
		return
	}
	if videoData.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized!", err)
		return
	}
	contentAndExtension := strings.Split(mediaType, "/")
	if len(contentAndExtension) != 2 || contentAndExtension[0] != "image" {
		respondWithError(w, http.StatusBadRequest, "Thumbnail has to be an image", err)
		return
	} 
	randFilename := make([]byte,32)
	rand.Read(randFilename)
	randFilenameStr := base64.URLEncoding.EncodeToString(randFilename)
	savePath := filepath.Join(cfg.assetsRoot,randFilenameStr + "." + contentAndExtension[1])
	save_file, err := os.Create(savePath)
	io.Copy(save_file,file)
	url := fmt.Sprintf("http://localhost:%s/assets/%s.%s",cfg.port,randFilenameStr ,contentAndExtension[1])
	videoData.ThumbnailURL = &url
	cfg.db.UpdateVideo(videoData)
	

	respondWithJSON(w, http.StatusOK, videoData)
}
