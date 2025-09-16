package main

import (
	"fmt"
	"os/exec"
)

func processVideoForFastStart(filepath string) (string, error){

	newFilePath := filepath + ".processing"

	cmd := exec.Command("ffmpeg",
						"-i",
						filepath,
						"-c",
						"copy",
						"-movflags",
						"faststart",
						"-f",
						"mp4",
						newFilePath)

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("Error in ffmpeg command: %w\n",err)
	}

	return newFilePath, nil
}