package worker

import (
	"context"
	"ecrhandler/internal/configs"
	"ecrhandler/internal/util"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/ecr/types"
	"log"
	"sync"
)

func RepoJob(id string, ctx context.Context, client *ecr.Client, wg *sync.WaitGroup, repoJobs chan string, imageJobs chan configs.EcrDeleteInputJob, counter *util.ImageJobCounting) {

	defer wg.Done()

	for repoName := range repoJobs {
		log.Printf("repo job %s working on repository %s\n", id, repoName)
		maxResult := int32(1000)
		listImageInput := &ecr.ListImagesInput{
			RepositoryName: &repoName,
			MaxResults:     &maxResult,
		}

		log.Printf("repo job %s list image parameter %+v \n", id, *listImageInput.RepositoryName)

		listImagesOutput, err := client.ListImages(ctx, listImageInput)
		util.CheckError(err)

		log.Printf("repo job %s list images images from repository %s found images %d\n", id, repoName, len(listImagesOutput.ImageIds))

		for _, imageItem := range listImagesOutput.ImageIds {
			log.Printf("repo job %s add to image channel %s %s\n", id, repoName, *imageItem.ImageDigest)
			counter.Inc(1)
			imageJobs <- configs.EcrDeleteInputJob{
				RepoName:      repoName,
				ImageDigestId: *imageItem.ImageDigest,
			}

		}
	}
}

func ImageJob(id string, ctx context.Context, client *ecr.Client, wg *sync.WaitGroup, imageJob chan configs.EcrDeleteInputJob, apiRequestRateLimit chan struct{}, imageJobCounter *util.ImageJobCounting) {

	for {
		_, okRate := <-apiRequestRateLimit
		if okRate {
			log.Printf("image job %s leaky bucket size %d\n", id, len(apiRequestRateLimit))
		}

		select {

		case ecrDeleteInputJob, ok := <-imageJob:
			if !ok {
				break
			}

			batchDeleteImageInput := &ecr.BatchDeleteImageInput{
				ImageIds: []types.ImageIdentifier{
					{
						ImageDigest: &ecrDeleteInputJob.ImageDigestId,
					},
				},
				RepositoryName: &ecrDeleteInputJob.RepoName,
			}

			batchDeleteImageOutput, err := client.BatchDeleteImage(ctx, batchDeleteImageInput)

			util.CheckError(err)

			if len(batchDeleteImageOutput.Failures) > 0 {

				failureImage := batchDeleteImageOutput.Failures[0]

				log.Printf("image job %s unable to delete image: %s, code: %s, messsage: %s\n", id, *failureImage.ImageId.ImageDigest, failureImage.FailureCode, *failureImage.FailureReason)
			} else {
				imageJobCounter.Inc(1)
				log.Printf("image job %s deleted ecr image, %s %s, %s \n", id, ecrDeleteInputJob.RepoName, ecrDeleteInputJob.ImageDigestId)
			}

		case <-ctx.Done():
			wg.Done()
			return
		}
	}
}
