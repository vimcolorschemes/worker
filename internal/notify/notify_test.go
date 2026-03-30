package notify

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sns"
)

type stubSNSClient struct {
	publish func(ctx context.Context, params *sns.PublishInput, optFns ...func(*sns.Options)) (*sns.PublishOutput, error)
}

func (s stubSNSClient) Publish(ctx context.Context, params *sns.PublishInput, optFns ...func(*sns.Options)) (*sns.PublishOutput, error) {
	return s.publish(ctx, params, optFns...)
}

func TestPublishJobNotification(t *testing.T) {
	originalLoadAWSConfig := loadAWSConfig
	originalNewSNSClient := newSNSClient
	t.Cleanup(func() {
		loadAWSConfig = originalLoadAWSConfig
		newSNSClient = originalNewSNSClient
		_ = os.Unsetenv("JOB_NOTIFICATIONS_TOPIC_ARN")
	})

	t.Run("publishes subject and message to sns", func(t *testing.T) {
		if err := os.Setenv("JOB_NOTIFICATIONS_TOPIC_ARN", "arn:aws:sns:us-east-1:123456789012:topic"); err != nil {
			t.Fatalf("Setenv: %v", err)
		}

		loadAWSConfig = func(ctx context.Context, optFns ...func(*awsconfig.LoadOptions) error) (aws.Config, error) {
			return aws.Config{}, nil
		}

		newSNSClient = func(cfg aws.Config) snsPublisher {
			return stubSNSClient{publish: func(ctx context.Context, params *sns.PublishInput, optFns ...func(*sns.Options)) (*sns.PublishOutput, error) {
				if got := aws.ToString(params.TopicArn); got != "arn:aws:sns:us-east-1:123456789012:topic" {
					t.Fatalf("TopicArn = %q, want topic arn", got)
				}
				if got := aws.ToString(params.Subject); got != "Daily Summary" {
					t.Fatalf("Subject = %q, want %q", got, "Daily Summary")
				}
				if got := aws.ToString(params.Message); got != "hello" {
					t.Fatalf("Message = %q, want %q", got, "hello")
				}
				return &sns.PublishOutput{}, nil
			}}
		}

		if err := PublishJobNotification(context.Background(), "Daily Summary", "hello"); err != nil {
			t.Fatalf("PublishJobNotification returned error: %v", err)
		}
	})

	t.Run("returns error when topic arn is missing", func(t *testing.T) {
		_ = os.Unsetenv("JOB_NOTIFICATIONS_TOPIC_ARN")

		err := PublishJobNotification(context.Background(), "Daily Summary", "hello")
		if err == nil {
			t.Fatal("PublishJobNotification error = nil, want error")
		}
	})

	t.Run("returns publish error", func(t *testing.T) {
		if err := os.Setenv("JOB_NOTIFICATIONS_TOPIC_ARN", "arn:aws:sns:us-east-1:123456789012:topic"); err != nil {
			t.Fatalf("Setenv: %v", err)
		}

		loadAWSConfig = func(ctx context.Context, optFns ...func(*awsconfig.LoadOptions) error) (aws.Config, error) {
			return aws.Config{}, nil
		}

		newSNSClient = func(cfg aws.Config) snsPublisher {
			return stubSNSClient{publish: func(ctx context.Context, params *sns.PublishInput, optFns ...func(*sns.Options)) (*sns.PublishOutput, error) {
				return nil, errors.New("boom")
			}}
		}

		err := PublishJobNotification(context.Background(), "Daily Summary", "hello")
		if err == nil {
			t.Fatal("PublishJobNotification error = nil, want error")
		}
	})
}
