package util

import (
	"log"
	"path/filepath"
)

func GetCurrentDir() string {
	dir, err := filepath.Abs(filepath.Dir("../../"))
	if err != nil {
		log.Fatal(err)
	}
	return dir
}
