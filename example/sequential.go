package main

import (
	"bufio"
	"context"
	"ecrhandler/internal/configs"
	"flag"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/ecr/types"
	"log"
	"slices"
	"time"

	"ecrhandler/internal/util"
	"os"
)

var slice_image = []string{"sha256:c5526f80371af594e173ce7ec0bfb1e9adff787953c5198f38e24b36870995c1"}

const RateLimit = 20

func newConfig() *configs.AppConfig {
	filePath := flag.String("f", "", "file path containing list of ecr repo")
	imageChanSize := flag.Int("r", 200, "buffer size of image channel")
	repoChanSize := flag.Int("i", 50, "buffer size repository channel")
	repoWorkerCount := flag.Int("iw", 1, "number of worker, default: no limit")
	imageWorkerCount := flag.Int("rw", 1, "number of worker, default: no limit")

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

func purgeImage(client *ecr.Client, ecrDeleteInputJob configs.EcrDeleteInputJob) {

	batchDeleteImageInput := &ecr.BatchDeleteImageInput{
		ImageIds: []types.ImageIdentifier{
			{
				ImageDigest: &ecrDeleteInputJob.ImageDigestId,
				ImageTag:    &ecrDeleteInputJob.ImageTag,
			},
		},
		RepositoryName: &ecrDeleteInputJob.RepoName,
	}

	batchDeleteImageOutput, err := client.BatchDeleteImage(context.TODO(), batchDeleteImageInput)

	util.CheckError(err)

	if len(batchDeleteImageOutput.Failures) > 0 {

		failureImage := batchDeleteImageOutput.Failures[0]

		log.Printf("unable to delete image: %s, code: %s, messsage: %s\n", *failureImage.ImageId.ImageDigest, failureImage.FailureCode, *failureImage.FailureReason)
	} else {
		log.Printf("deleted ecr image, %s %s, %s \n", ecrDeleteInputJob.RepoName, ecrDeleteInputJob.ImageDigestId, ecrDeleteInputJob.ImageTag)
	}

}

func processRepo(client *ecr.Client, repoName string) {
	log.Printf("working on repository %s\n", repoName)
	maxResult := int32(1000)
	listImageInput := &ecr.ListImagesInput{
		RepositoryName: &repoName,
		MaxResults:     &maxResult,
	}

	log.Printf("list image parameter %+v \n", *listImageInput.RepositoryName)

	listImagesOutput, err := client.ListImages(context.TODO(), listImageInput)
	util.CheckError(err)

	log.Printf("list images images from repository %s found images %d\n", repoName, len(listImagesOutput.ImageIds))

	for _, imageItem := range listImagesOutput.ImageIds {
		if slices.Contains(slice_image, *imageItem.ImageDigest) {
			imageConfig := configs.EcrDeleteInputJob{
				RepoName:      repoName,
				ImageDigestId: *imageItem.ImageDigest,
				ImageTag:      *imageItem.ImageTag,
			}

			purgeImage(client, imageConfig)
		}

	}
}

func timer(name string) func() {
	start := time.Now()
	return func() {
		fmt.Printf("%s took %v\n", name, time.Since(start))
	}
}

func newEcrClient() *ecr.Client {

	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRetryer(func() aws.Retryer {
		return retry.AddWithMaxBackoffDelay(retry.NewAdaptiveMode(), time.Second*2)
	}))
	util.CheckError(err)

	return ecr.NewFromConfig(cfg)
}

func main() {

	defer timer("main")()
	//ctx, cancel := context.WithCancel(context.Background())

	//defer cancel()
	//wg := &sync.WaitGroup{}
	appConfig := newConfig()

	//leakyBucket := make(chan struct{}, RateLimit)

	//wg.Add(1)
	//go func() {
	//	ticker := time.NewTicker(time.Second / RateLimit)
	//	for {
	//		select {
	//		case <-ticker.C:
	//			if len(leakyBucket) < RateLimit {
	//				leakyBucket <- struct{}{}
	//			}
	//		case <-ctx.Done():
	//			fmt.Println("rate limit ticker context done")
	//			wg.Done()
	//			close(leakyBucket)
	//			return
	//		}
	//	}
	//}()

	file, err := os.Open(appConfig.FilePath)
	util.CheckError(err)

	ecrClient := newEcrClient()

	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		repoName := scanner.Text()
		processRepo(ecrClient, repoName)
	}

	//wg.Wait()

	log.Println("operation completed")
}
