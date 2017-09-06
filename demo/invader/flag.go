package main

import (
	"log"

	"golang.org/x/mobile/asset"
)

func flagBool(value *bool, name string) {
	if exists(name) {
		*value = true
	}
	log.Printf("flagBool: %s = %v", name, *value)
}

func flagStr(value *string, name string) error {
	b, errLoad := loadFull(name)
	if errLoad != nil {
		log.Printf("flagStr: %s: %v", name, errLoad)
		return errLoad
	}
	*value = string(b)
	log.Printf("flagStr: %s = %v", name, *value)
	return nil
}

func exists(name string) bool {
	f, errOpen := asset.Open(name)
	if errOpen != nil {
		return false
	}
	f.Close()
	return true
}
