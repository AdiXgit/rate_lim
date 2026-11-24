package main

import "log"

func main() {
	cfg, err := LoadSettings()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("loaded settings: %+v", cfg)
}
