package storage

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/ec2rolecreds"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecr"
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
	creds := credentials.NewChainCredentials([]credentials.Provider{
		&credentials.StaticProvider{
			Value: credentials.Value{
				AccessKeyID:     accessKey,
				SecretAccessKey: secretKey,
			},
		},
		&credentials.EnvProvider{},
		&credentials.SharedCredentialsProvider{},
		&ec2rolecreds.EC2RoleProvider{Client: ec2metadata.New(session.New())},
	})
	awsConfig := aws.NewConfig()
	awsConfig.WithCredentials(creds)
	awsConfig.WithRegion(region)

	session, err := session.NewSession(awsConfig)
	if err != nil {
		return err
	}
	repoInput := &ecr.CreateRepositoryInput{
		RepositoryName: aws.String(reponame),
	}
	if _, err := ecr.New(session).CreateRepository(repoInput); err != nil {
		if s3Err, ok := err.(awserr.Error); ok && s3Err.Code() == "RepositoryAlreadyExistsException" {
			return nil
		}
		return err
	}
	return nil
}
