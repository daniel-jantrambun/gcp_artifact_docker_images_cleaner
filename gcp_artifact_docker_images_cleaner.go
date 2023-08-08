package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/peterbourgon/ff/v3"
	log "github.com/sirupsen/logrus"

	"cloud.google.com/go/artifactregistry/apiv1"
	"cloud.google.com/go/artifactregistry/apiv1/artifactregistrypb"
	"google.golang.org/api/iterator"
)

func main() {
	log.SetFormatter(&log.JSONFormatter{})
	log.Info("start artifact registry management")
	fs := flag.NewFlagSet("gcp_artifact_docker_images_cleaner", flag.ContinueOnError)
	var (
		project      = fs.String("project", "unique-functions", "GCP project id (env PROJECT)")
		location     = fs.String("location", "europe-west4", "artifact registry location (env LOCATION)")
		repository   = fs.String("repository", "", "repository name")
		nbDaysToKeep = fs.Int("days-to-keep", 60, "nb days to keep docker images")
		dry          = fs.Bool("dry", false, "dry run")
		concurrency  = fs.Int("concurrency", 5, "concurrency")
	)
	if err := ff.Parse(fs, os.Args[1:], ff.WithEnvVars()); err != nil {
		log.Errorf("error: %v\n", err)
		os.Exit(1)
	}
	log.Printf("project: %s\n", *project)
	log.Infof("location: %s\n", *location)
	log.Infof("repository: %s\n", *repository)
	log.Infof("nbDaysToKeep: %d\n", *nbDaysToKeep)
	log.Infof("dry: %t\n", *dry)
	log.Infof("concurrency: %d\n", *concurrency)

	ctx := context.Background()
	c, err := artifactregistry.NewClient(ctx)
	if err != nil {
		log.Errorf("error creating artifact registry client: %v\n", err)
		os.Exit(1)
	}
	defer c.Close()

	referenceDate := time.Now().AddDate(0, 0, -(int(*nbDaysToKeep)))
	log.Infof("will delete docker images older than %s or without tag\n", referenceDate)

	req := &artifactregistrypb.ListDockerImagesRequest{
		Parent:  fmt.Sprintf("projects/%s/locations/%s/repositories/%s", *project, *location, *repository),
		OrderBy: "build_time desc",
	}
	it := c.ListDockerImages(ctx, req)

	imagesToDelete := make([]string, 0)
	tagsToDelete := make([]string, 0)

	for {
		resp, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Errorf("error iterating over docker images: %v\n", err)
			os.Exit(1)
		}
		toDelete := false

		if len(resp.Tags) == 0 {
			toDelete = true
		} else {
			if resp.UploadTime.AsTime().Before(referenceDate) {
				for _, tag := range resp.Tags {
					if strings.Contains(tag, "latest") {
						toDelete = false
						break
					}
					toDelete = true
				}
			}
		}

		if toDelete {
			parts := strings.Split(resp.Name, "/")
			imageName := strings.Split(parts[len(parts)-1], "@")
			packageName := imageName[0]
			version := imageName[1]

			log.Infof("Image will be deleted: %s/%s\n", packageName, version)
			versionName := fmt.Sprintf("projects/%s/locations/%s/repositories/%s/packages/%s/versions/%s", *project, *location, *repository, packageName, version)
			for _, tag := range resp.Tags {
				log.Infof("Tag to be deleted for this image: %s\n", tag)
				tagName := fmt.Sprintf("projects/%s/locations/%s/repositories/%s/packages/%s/tags/%s", *project, *location, *repository, packageName, tag)
				tagsToDelete = append(tagsToDelete, tagName)
			}

			imagesToDelete = append(imagesToDelete, versionName)
		}
	}

	if !*dry {
		log.Info("will delete tags and images")
		var wg sync.WaitGroup
		log.Infof("will delete %d docker tags\n", len(tagsToDelete))
		number := *concurrency
		if len(tagsToDelete) < number {
			number = len(tagsToDelete)
		}
		for i := 0; i < number; i++ {
			wg.Add(1)
			go func(j int) {
				defer wg.Done()
				deleteTags(ctx, c, tagsToDelete[j*len(tagsToDelete)/number:(j+1)*len(tagsToDelete)/number], j)
			}(i)
		}
		wg.Wait()

		log.Infof("will delete %d docker images\n", len(imagesToDelete))
		number = *concurrency
		if len(imagesToDelete) < *concurrency {
			number = len(imagesToDelete)
		}
		for i := 0; i < number; i++ {
			wg.Add(1)
			go func(j int) {
				defer wg.Done()
				deleteImages(ctx, c, imagesToDelete[j*len(imagesToDelete)/number:(j+1)*len(imagesToDelete)/number], j)
			}(i)
		}
		wg.Wait()
	} else {
		log.Info("dry run, will not delete anything")
	}

	log.Info("end artifact registry management")
}

func deleteTags(ctx context.Context, c *artifactregistry.Client, tagsToDelete []string, i int) error {
	log.Infof("start deleting %d tags for group %d\n", len(tagsToDelete), i)
	for _, tagName := range tagsToDelete {
		// log.Infof("tag name: %s\n", tagName)
		req := &artifactregistrypb.DeleteTagRequest{
			Name: tagName,
		}
		err := c.DeleteTag(ctx, req)
		if err != nil {
			log.Errorf("error deleting docker image tag: %v\n", err)
		}
		// log.Info("tag deleted")
	}

	log.Infof("end tags deleting for group %d\n", i)
	return nil
}

func deleteImages(ctx context.Context, c *artifactregistry.Client, imagesToDelete []string, i int) error {
	log.Infof("start deleting %d images for group %d\n", len(imagesToDelete), i)
	for _, imageName := range imagesToDelete {
		req := &artifactregistrypb.DeleteVersionRequest{
			Name: imageName,
		}
		op, err := c.DeleteVersion(ctx, req)
		if err != nil {
			log.Infof("error deleting docker image: %v\n", err)
		}
		err = op.Wait(ctx)
		if err != nil {
			log.Errorf("error waiting for delete docker image operation: %v\n", err)
		}
	}
	log.Infof("end images deleting for group %d\n", i)
	return nil
}
