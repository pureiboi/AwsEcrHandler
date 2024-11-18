package configs

import "log"

func (c *AppConfig) ValidateConfig() {

	if c.FilePath == "" {
		log.Panic("file is expected")
	}

	if c.RepoWorkerCount <= 0 {
		log.Panic("no. of worker for repository is expected to be more than 0")
	}

	if c.ImageWorkerCount <= 0 {
		log.Panic("no. of worker for image is expected to be more than 0")
	}

}
