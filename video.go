package main

import (
	"encoding/json"
	"os/exec"
	"bytes"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/database"
	"time"
	"context"
	"strings"
)

func getVideoAspectRatio(filePath string) (string, error) {
	newCmd := exec.Command("ffprobe","-v","error","-print_format","json","-show_streams",filePath)
	var byteBuffer bytes.Buffer 
	newCmd.Stdout = &byteBuffer
	newCmd.Run()

	type AspectRatioJson struct  {
		Streams []struct{
			Width	int		`json:"width"`
			Height	int		`json:"height"`

		} `json:"streams"`
	}
	var aspectRatio AspectRatioJson

	err := json.Unmarshal(byteBuffer.Bytes(),&aspectRatio)
	if err != nil{
		return "", err
	}
	ratio := float64(aspectRatio.Streams[0].Width) / float64(aspectRatio.Streams[0].Height)
	if ratio < 1.8 && ratio >1.6{
		return "landscape",nil
	}
	if ratio < 0.6 && ratio>0.5 {
		return "portrait", nil
	} else {
		return "other", nil
	}
}

func processVideoForFastStart(filePath string) (string, error) {
	outpath := filePath + ".processing"
	newCmd := exec.Command("ffmpeg","-i",filePath,"-c","copy","-movflags","faststart","-f","mp4",outpath)
	err := newCmd.Run()
	if err != nil{
		return "", err
	}
	return outpath, nil
}

func generatePresignedURL(s3Client *s3.Client, bucket, key string, expireTime time.Duration) (string, error) {
	presign_s3Client := s3.NewPresignClient(s3Client)
	presignObjectInput := &s3.GetObjectInput{
		Bucket: &bucket,
		Key:	&key,
	}
	result, err := presign_s3Client.PresignGetObject(context.TODO(),presignObjectInput,s3.WithPresignExpires(expireTime))
	if err != nil {
		return "", err
	}

	return result.URL, nil
}

func (cfg *apiConfig) dbVideoToSignedVideo(video database.Video) (database.Video, error) {
	if video.VideoURL == nil{
		return video,nil
	}
	bucketAndKey := strings.Split(*video.VideoURL,";")
	new_url, err := generatePresignedURL(cfg.s3Client,bucketAndKey[0],bucketAndKey[1],5*time.Minute)
	if err != nil {
		return database.Video{},err
	}
	video.VideoURL = &new_url
	return video, nil
}