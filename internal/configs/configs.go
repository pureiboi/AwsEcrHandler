package configs

type EcrDeleteInputJob struct {
	RepoName      string
	ImageDigestId string
	ImageTag      string
}

type AppConfig struct {
	FilePath         string
	ImageChanSize    int
	RepoChanSize     int
	RepoWorkerCount  int
	ImageWorkerCount int
}
