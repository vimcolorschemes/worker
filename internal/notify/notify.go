package notify

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sns"
)

var loadAWSConfig = awsconfig.LoadDefaultConfig

type snsPublisher interface {
	Publish(ctx context.Context, params *sns.PublishInput, optFns ...func(*sns.Options)) (*sns.PublishOutput, error)
}

var newSNSClient = func(cfg aws.Config) snsPublisher {
	return sns.NewFromConfig(cfg)
}

// PublishJobNotification sends a plain-text job notification email through SNS.
func PublishJobNotification(ctx context.Context, subject string, body string) error {
	topicARN := strings.TrimSpace(os.Getenv("JOB_NOTIFICATIONS_TOPIC_ARN"))
	if topicARN == "" {
		return fmt.Errorf("JOB_NOTIFICATIONS_TOPIC_ARN not found in env")
	}

	cfg, err := loadAWSConfig(ctx)
	if err != nil {
		return fmt.Errorf("load aws config: %w", err)
	}

	_, err = newSNSClient(cfg).Publish(ctx, &sns.PublishInput{
		TopicArn: aws.String(topicARN),
		Subject:  aws.String(subject),
		Message:  aws.String(body),
	})
	if err != nil {
		return fmt.Errorf("publish sns notification: %w", err)
	}

	return nil
}
