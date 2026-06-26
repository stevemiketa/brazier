package secrets

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

// AWSSSMBackend reads and writes secrets via AWS Systems Manager Parameter Store.
type AWSSSMBackend struct {
	client *ssm.Client
	prefix string // path prefix for all parameters, e.g. "/brazier/prod"
}

// NewAWSSSMBackend creates an AWSSSMBackend using the default AWS credential chain.
func NewAWSSSMBackend(ctx context.Context, prefix string) (*AWSSSMBackend, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}
	return &AWSSSMBackend{client: ssm.NewFromConfig(cfg), prefix: prefix}, nil
}

func (a *AWSSSMBackend) paramName(name string) string {
	return a.prefix + "/" + name
}

func (a *AWSSSMBackend) Get(ctx context.Context, name string) (string, error) {
	out, err := a.client.GetParameter(ctx, &ssm.GetParameterInput{
		Name:           aws.String(a.paramName(name)),
		WithDecryption: aws.Bool(true),
	})
	if err != nil {
		return "", fmt.Errorf("ssm get %s: %w", name, err)
	}
	return aws.ToString(out.Parameter.Value), nil
}

func (a *AWSSSMBackend) Set(ctx context.Context, name, value string) error {
	_, err := a.client.PutParameter(ctx, &ssm.PutParameterInput{
		Name:      aws.String(a.paramName(name)),
		Value:     aws.String(value),
		Type:      "SecureString",
		Overwrite: aws.Bool(true),
	})
	if err != nil {
		return fmt.Errorf("ssm put %s: %w", name, err)
	}
	return nil
}

func (a *AWSSSMBackend) Delete(ctx context.Context, name string) error {
	_, err := a.client.DeleteParameter(ctx, &ssm.DeleteParameterInput{
		Name: aws.String(a.paramName(name)),
	})
	if err != nil {
		return fmt.Errorf("ssm delete %s: %w", name, err)
	}
	return nil
}
