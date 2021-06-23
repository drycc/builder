package sshd

import (
	"time"
)

// Config represents the required SSH server configuration.
type Config struct {
	ControllerHost              string `envconfig:"DRYCC_CONTROLLER_SERVICE_HOST" required:"true"`
	ControllerPort              string `envconfig:"DRYCC_CONTROLLER_SERVICE_PORT" required:"true"`
	SSHHostIP                   string `envconfig:"SSH_HOST_IP" default:"0.0.0.0" required:"true"`
	SSHHostPort                 int    `envconfig:"SSH_HOST_PORT" default:"2223" required:"true"`
	HealthSrvPort               int    `envconfig:"HEALTH_SERVER_PORT" default:"8092"`
	HealthSrvTestStorageRegion  string `envconfig:"STORAGE_REGION" default:"us-east-1"`
	CleanerPollSleepDurationSec int    `envconfig:"CLEANER_POLL_SLEEP_DURATION_SEC" default:"5"`
	StorageType                 string `envconfig:"BUILDER_STORAGE" default:"minio"`
	BuildpackerImagePullPolicy  string `envconfig:"BUILDPACKER_IMAGE_PULL_POLICY" default:"Always"`
	ImagebuilderImagePullPolicy string `envconfig:"IMAGEBUILDER_IMAGE_PULL_POLICY" default:"Always"`
	LockTimeout                 int    `envconfig:"GIT_LOCK_TIMEOUT" default:"10"`
}

// CleanerPollSleepDuration returns c.CleanerPollSleepDurationSec as a time.Duration.
func (c Config) CleanerPollSleepDuration() time.Duration {
	return time.Duration(c.CleanerPollSleepDurationSec) * time.Second
}

//GitLockTimeout return LockTimeout in minutes
func (c Config) GitLockTimeout() time.Duration {
	return time.Duration(c.LockTimeout) * time.Minute
}
