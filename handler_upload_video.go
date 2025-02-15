package main

import (
	"fmt"
	"net/http"
	"io"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
	"os"
	"mime"
	"context"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"crypto/rand"
	"encoding/base64"

)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {

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


	fmt.Println("uploading video ", videoID, "by user", userID)

	const maxMemory = 10 << 30
	r.ParseMultipartForm(maxMemory)

	file, header, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return
	}
	defer file.Close()

	videoData, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Failed to fetch Metadata", err)
		return
	}
	if videoData.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized!", err)
		return
	}

	contentType := header.Header.Get("Content-Type")
	mediatype, params, err := mime.ParseMediaType(contentType)
	if mediatype != "video/mp4"{
		respondWithError(w, http.StatusBadRequest, "Upload only allowed for mp4 videos!", err)
		return
	}
	fmt.Println(mediatype)
	for k := range params {
		fmt.Println(params[k])
	}
	tempFile, err := os.CreateTemp("","tubely-upload.mp4")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Failed to upload video!", err)
		return
	}
	defer os.Remove(tempFile.Name())
	_, err = io.Copy(tempFile,file)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Failed to save video!", err)
		return
	}
	_, err = tempFile.Seek(0, io.SeekStart)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Failed to read uploaded video!", err)
		return
	}
	aspectRatioStr, err := getVideoAspectRatio(tempFile.Name())
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Failed to get Aspect Ratio!", err)
		return
	}
	fastStartPath, err  := processVideoForFastStart(tempFile.Name())
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Failed to convert video to FastStartFormat!", err)
		return
	}
	fastStartFile, err := os.Open(fastStartPath)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Failed to open converted FastStartVideo!", err)
		return
	}
	defer os.Remove(fastStartPath)
	randFilename := make([]byte,32)
	rand.Read(randFilename)
	randFilenameStr := base64.URLEncoding.EncodeToString(randFilename)
	randFilenameStr = aspectRatioStr + "/" + randFilenameStr + ".mp4"
	_, err = cfg.s3Client.PutObject(context.TODO(),&s3.PutObjectInput{
		Bucket: &cfg.s3Bucket,
		Key: &randFilenameStr,
		ContentType: &contentType,
		Body: fastStartFile,

	})

	if err != nil{
		respondWithError(w, http.StatusBadRequest, "Failed to mirror uploaded video on s3 webserver!", err)
		return

	}

	url := fmt.Sprintf("%s;%s",cfg.s3Bucket,randFilenameStr)
	videoData.VideoURL = &url
	cfg.db.UpdateVideo(videoData)
	retVideo,err := cfg.dbVideoToSignedVideo(videoData)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Failed to generate Video URL!", err)
		return
	}

	respondWithJSON(w, http.StatusOK, retVideo.VideoURL)

}
