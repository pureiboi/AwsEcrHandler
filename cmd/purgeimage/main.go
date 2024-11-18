package main

import (
	"bufio"
	"context"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"log"

	"fmt"
	"os"
	"sync"

	"ecrhandler/internal/util"
	"ecrhandler/internal/worker"
)

func newEcrClient() *ecr.Client {

	cfg, err := config.LoadDefaultConfig(context.TODO())
	util.CheckError(err)

	return ecr.NewFromConfig(cfg)
}

func main() {

	config := newConfig()

	fmt.Println(config)

	repoJobs := make(chan string, config.repoChanSize)
	imageJobs := make(chan string, config.imageChanSize)

	wg := &sync.WaitGroup{}

	file, err := os.Open(config.filePath)
	ecrClient := newEcrClient()

	util.CheckError(err)

	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		wg.Add(1)
		repoName := scanner.Text()
		log.Printf("spawning goroutine process repository %s\n", repoName)
		go worker.WorkerRepo(ecrClient, wg, repoJobs, imageJobs, imageJobs)
		repoJobs <- repoName
	}
	wg.Wait()

	close(repoJobs)
	close(imageJobs)
}
