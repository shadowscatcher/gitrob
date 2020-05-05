package _testing_support

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

var contents [][]byte

func readFile(path string, info os.FileInfo, err error) error {
	if info.IsDir() {
		return nil
	}
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	contents = append(contents, data)
	return nil
}

func ReadFiles(path string) [][]byte {
	err := filepath.Walk(path, readFile)
	if err != nil {
		log.Panicf("Failed reading file: %s", err)
	}
	return contents
}