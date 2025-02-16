package storage

import (
	"log"
	"os"
	"tebexpressapi/pkg/config"
	"testing"
)

func TestBackBlazeUpload(t *testing.T) {
	err := config.ReadConfigByFiles("toml", []string{"../../config_dev.toml"})
	if err != nil {
		t.Log(err)
	}

	// file, err := OpenFileAsReader("../../config_dev.toml")

	// Open the file
	file, err := os.Open("../../config_dev.toml")
	if err != nil {
		t.Log(err)
	}

	fileInfo, err := file.Stat()
	if err != nil {
		log.Fatalf("Error getting file info: %v", err)
	}

	log.Println(err)
	log.Println(fileInfo.Size())
	b2 := NewBackblaze(nil)
	err = b2.UploadFile(file, "config_dev.toml", fileInfo.Size(), "nde2024", "")
	log.Println(err)
}

func TestBackBlazeGet(t *testing.T) {
	err := config.ReadConfigByFiles("toml", []string{"../../config_dev.toml"})
	if err != nil {
		t.Log(err)
	}

	b2 := NewBackblaze(nil)
	_, _, err = b2.Read("config_dev.toml", "nde2024")
	log.Println(err)
}
