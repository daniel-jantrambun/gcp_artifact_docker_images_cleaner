package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
    "sync"

	"cloud.google.com/go/artifactregistry/apiv1"
	"cloud.google.com/go/artifactregistry/apiv1/artifactregistrypb"
	"google.golang.org/api/iterator"
)

var (
	project      = flag.String("project", "", "GCP project id")
	location     = flag.String("location", "europe-west4", "artifact registry location")
	repository   = flag.String("repository", "", "repository name")
	nbDaysToKeep = flag.Int("daysToKeep", 30, "nb days to keep docker images")
	dry          = flag.Bool("dry", false, "dry run")
)

func main() {
	fmt.Println("start artifact registry management")
	err := initVariables()
	if err != nil {
		fmt.Printf("error getting args: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	c, err := artifactregistry.NewClient(ctx)
	if err != nil {
		fmt.Printf("error creating artifact registry client: %v\n", err)
		os.Exit(1)
	}
	defer c.Close()

	referenceDate := time.Now().AddDate(0, 0, -(int(*nbDaysToKeep)))
	fmt.Printf("will delete docker images older than %s or without tag\n", referenceDate)

	req := &artifactregistrypb.ListDockerImagesRequest{
		Parent:   fmt.Sprintf("projects/%s/locations/%s/repositories/%s", *project, *location, *repository),
		OrderBy:  "build_time desc",
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
			fmt.Printf("error iterating over docker images: %v\n", err)
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

			versionName := fmt.Sprintf("projects/%s/locations/%s/repositories/%s/packages/%s/versions/%s", *project, *location, *repository, packageName, version)
			for _, tag := range resp.Tags {
				fmt.Printf("tag: %s\n", tag)
				tagName := fmt.Sprintf("projects/%s/locations/%s/repositories/%s/packages/%s/tags/%s", *project, *location, *repository, packageName, tag)
				tagsToDelete = append(tagsToDelete, tagName)
			}

			imagesToDelete = append(imagesToDelete, versionName)
		}
	}

	if !*dry {
		fmt.Println("will delete tags and images")
		var wg sync.WaitGroup
		fmt.Printf("will delete %d docker tags\n", len(tagsToDelete))
		number := 5
		if len(tagsToDelete) < 5 {number = len(tagsToDelete)}
		for i := 0; i < number; i++ {
			wg.Add(1)
			go func(j int) {
				defer wg.Done()
				deleteTags(ctx, c, tagsToDelete[j*len(tagsToDelete)/number:(j+1)*len(tagsToDelete)/number], j)
			}(i)
		}
		wg.Wait()

		fmt.Printf("will delete %d docker images\n", len(imagesToDelete))
		number = 5
		if len(imagesToDelete) < 5 {number = len(imagesToDelete)}
		for i := 0; i < number ; i++ {
			wg.Add(1)
			go func(j int) {
				defer wg.Done()
				deleteImages(ctx, c, imagesToDelete[j*len(imagesToDelete)/number:(j+1)*len(imagesToDelete)/number], j)
			}(i)
		}
		wg.Wait()
	}

	fmt.Println("end artifact registry management")
}

func deleteTags(ctx context.Context, c *artifactregistry.Client, tagsToDelete []string, i int) error {
	// fmt.Printf("start deleting %d tags for group %d\n", len(tagsToDelete), i)
	for _, tagName := range tagsToDelete {
		// fmt.Printf("tag name: %s\n", tagName)
		req := &artifactregistrypb.DeleteTagRequest{
			Name: tagName,
		}
		err := c.DeleteTag(ctx, req)
		if err != nil {
			fmt.Printf("error deleting docker image tag: %v\n", err)
		}
		// fmt.Println("tag deleted")
	}

	// fmt.Printf("end tags deleting for group %d\n", i)
	return nil
}

func deleteImages(ctx context.Context, c *artifactregistry.Client, imagesToDelete []string, i int) error {
	// fmt.Printf("start deleting %d images for group %d\n", len(imagesToDelete), i)
	for _, imageName := range imagesToDelete {
		req := &artifactregistrypb.DeleteVersionRequest{
			Name: imageName,
		}
		op, err := c.DeleteVersion(ctx, req)
		if err != nil {
			fmt.Printf("error deleting docker image: %v\n", err)
		}
		err = op.Wait(ctx)
		if err != nil {
			fmt.Printf("error waiting for delete docker image operation: %v\n", err)
		}
	}
	fmt.Printf("end images deleting for group %d\n", i)
	return nil
}

func initVariables() error {
	flag.Parse()
	if *repository == "" {
		return errors.New("repository has no value")
	}
	return nil
}
