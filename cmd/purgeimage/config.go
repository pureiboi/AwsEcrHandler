package main

import (
	"flag"
	"log"
)

type appConfig struct {
	filePath      string
	imageChanSize int
	repoChanSize  int
}

func validateConfig(config appConfig){

	if config.filePath == "" {
		log.Panic("file is expected")
	}

}

func newConfig() appConfig {
	filePath := flag.String("f", "", "file path containing list of ecr repo")
	imageChanSize := flag.Int("r", 100, "buffer size of image channel")
	repoChanSize := flag.Int("i", 50, "buffer size repository channel")

	flag.Parse()

	config := appConfig{
		filePath:      *filePath,
		imageChanSize: *imageChanSize,
		repoChanSize:  *repoChanSize,
	}
	
	validateConfig(config)

	return config
}


