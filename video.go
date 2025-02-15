package main

import (
	"encoding/json"
	"os/exec"
	"bytes"
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