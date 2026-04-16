// Package storage provides image repository management functionality.
package storage

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/ecr/types"
)

// CreateImageRepo create a repository for the image on amazon's ECR(EC2 Container Repository)
// if it doesn't exist as repository needs to be present before pushing and image into it.
func CreateImageRepo(reponame string, params map[string]string) error {
	var (
		accessKey  string
		secretKey  string
		regionName string
		ok         bool
	)

	accessKey, ok = params["accesskey"]
	if !ok {
		accessKey = ""
	}
	secretKey, ok = params["secretkey"]
	if !ok {
		secretKey = ""
	}
	regionName, ok = params["region"]
	if !ok || fmt.Sprint(regionName) == "" {
		return fmt.Errorf("no region parameter provided: %s", regionName)
	}
	region := fmt.Sprint(regionName)

	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(region),
		config.WithCredentialsProvider(
			aws.NewCredentialsCache(
				credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
			),
		),
	)
	if err != nil {
		return err
	}

	client := ecr.NewFromConfig(cfg)
	repoInput := &ecr.CreateRepositoryInput{
		RepositoryName: aws.String(reponame),
	}
	if _, err := client.CreateRepository(ctx, repoInput); err != nil {
		var repoExistsErr *types.RepositoryAlreadyExistsException
		if ok := errors.As(err, &repoExistsErr); ok {
			return nil
		}
		return err
	}
	return nil
}
