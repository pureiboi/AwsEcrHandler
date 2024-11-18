package worker

import (
	"context"
	"ecrhandler/internal/util"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/ecr/types"
	"log"
	"sync"
	"time"
)

func workerImage(client *ecr.Client, wg *sync.WaitGroup, repoName string, imageJobRec <-chan string, imageJobSend chan<- string) {
	time.Sleep(800 * time.Millisecond) // to avoid api throttling
	defer wg.Done()
	imageDigestId, more := <-imageJobRec
	if more {
		batchDeleteImageInput := &ecr.BatchDeleteImageInput{
			ImageIds: []types.ImageIdentifier{
				{
					ImageDigest: &imageDigestId,
				},
			},
			RepositoryName: &repoName,
		}

		batchDeleteImageOutput, err := client.BatchDeleteImage(context.TODO(), batchDeleteImageInput)

		util.CheckError(err)

		if len(batchDeleteImageOutput.Failures) > 0 {

			failureImage := batchDeleteImageOutput.Failures[0]

			// retry for failed images possibly referenced by another image
			wg.Add(1)
			go workerImage(client, wg, repoName, imageJobRec, imageJobSend)
			imageJobSend <- imageDigestId

			log.Printf("unable to delete image: %s, code: %s, messsage: %s", *failureImage.ImageId.ImageDigest, failureImage.FailureCode, *failureImage.FailureReason)
		} else {
			fmt.Println("deleted ecr image", imageDigestId)
		}
	}
}

func WorkerRepo(client *ecr.Client, wg *sync.WaitGroup, repoJobs <-chan string, imageJobRec chan<- string, imageJobSend <-chan string) {
	defer wg.Done()
	repoName, more := <-repoJobs
	if more {

		listImageInput := &ecr.ListImagesInput{
			RepositoryName: &repoName,
		}

		listImagesOutput, err := client.ListImages(context.TODO(), listImageInput)

		util.CheckError(err)

		log.Printf("list images images from repo %s\n", repoName)
		log.Printf("spawning goroutine to purge images for repository%s\n", repoName)

		for _, imageItem := range listImagesOutput.ImageIds {

			wg.Add(1)
			go workerImage(client, wg, repoName, imageJobSend, imageJobRec)
			imageJobRec <- *imageItem.ImageDigest

		}

	}
}
