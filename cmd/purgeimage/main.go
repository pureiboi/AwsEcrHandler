package main

import (
	"bufio"
	"context"
	"ecrhandler/internal/configs"
	"flag"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"log"
	"strconv"
	"time"

	"os"
	"sync"

	"ecrhandler/internal/util"
	"ecrhandler/internal/worker"
)

const RateLimit = 20

func newConfig() *configs.AppConfig {
	filePath := flag.String("f", "", "file path containing list of ecr repo")
	imageChanSize := flag.Int("r", 200, "buffer size of image channel")
	repoChanSize := flag.Int("i", 50, "buffer size repository channel")
	repoWorkerCount := flag.Int("iw", 2, "number of worker for repository")
	imageWorkerCount := flag.Int("rw", 10, "number of worker for image")

	flag.Parse()

	appConfig := &configs.AppConfig{
		FilePath:         *filePath,
		ImageChanSize:    *imageChanSize,
		RepoChanSize:     *repoChanSize,
		RepoWorkerCount:  *repoWorkerCount,
		ImageWorkerCount: *imageWorkerCount,
	}

	appConfig.ValidateConfig()

	return appConfig
}

func timer(name string) func() {
	start := time.Now()
	return func() {
		fmt.Printf("%s took %v\n", name, time.Since(start))
	}
}

func newEcrClient() *ecr.Client {

	cfg, err := config.LoadDefaultConfig(context.TODO())
	util.CheckError(err)

	return ecr.NewFromConfig(cfg)
}

func main() {

	defer timer("main")()
	ctx, cancel := context.WithCancel(context.Background())

	defer cancel()
	wg := &sync.WaitGroup{}
	appConfig := newConfig()

	mu := &sync.Mutex{}

	imageJobTotal := &util.ImageJobCounting{
		Counter: 0,
		Mu:      mu,
	}

	imageJobCounter := &util.ImageJobCounting{
		Counter: 0,
		Mu:      mu,
	}

	repoJobs := make(chan string, appConfig.RepoChanSize)
	imageJobs := make(chan configs.EcrDeleteInputJob, appConfig.ImageChanSize)

	leakyBucket := make(chan struct{}, RateLimit)

	wg.Add(1)
	go func() {
		ticker := time.NewTicker(time.Second / RateLimit)
		for {
			select {
			case <-ticker.C:
				if len(leakyBucket) < RateLimit {
					leakyBucket <- struct{}{}
				}
			case <-ctx.Done():
				log.Println("rate limit ticker context done")
				wg.Done()
				close(leakyBucket)
				return
			}
		}
	}()

	wg.Add(1)
	go func() {
		time.Sleep(time.Second * 5)
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			default:
				time.Sleep(time.Second / 4)
				if imageJobTotal.Counter == imageJobCounter.Counter {
					cancel()
				}
			}
		}
	}()

	file, err := os.Open(appConfig.FilePath)
	util.CheckError(err)

	ecrClient := newEcrClient()

	defer file.Close()

	scanner := bufio.NewScanner(file)

	log.Printf("spawning %d goroutine repository worker pool\n", appConfig.RepoWorkerCount)
	wg.Add(appConfig.RepoWorkerCount)
	for x := range appConfig.RepoWorkerCount {
		go worker.RepoJob(strconv.Itoa(x), ctx, ecrClient, wg, repoJobs, imageJobs, imageJobTotal)
	}

	log.Printf("spawning %d goroutine image worker pool\n", appConfig.ImageWorkerCount)
	wg.Add(appConfig.ImageWorkerCount)
	for x := range appConfig.ImageWorkerCount {
		go worker.ImageJob(strconv.Itoa(x), ctx, ecrClient, wg, imageJobs, leakyBucket, imageJobCounter)
	}

	for scanner.Scan() {
		repoName := scanner.Text()
		repoJobs <- repoName
	}

	close(repoJobs)
	wg.Wait()
	close(imageJobs)

	log.Println("operation completed")
}
