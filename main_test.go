package main

import (
	"io/ioutil"
	"testing"
)

func TestMain(t *testing.T) {
	path := "main.go"
	_, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatalf("Unable to read file: %v", err)
	}
}
