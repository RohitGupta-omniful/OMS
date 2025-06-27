package SQS

import (
	"context"

	"github.com/omniful/go_commons/config"
	"github.com/omniful/go_commons/i18n"
	"github.com/omniful/go_commons/log"
	"github.com/omniful/go_commons/sqs"
)

func PublishCreateBulkOrderEvent(ctx context.Context) (*sqs.Publisher, error) {
	sqsCfg := &sqs.Config{
		Account:  config.GetString(ctx, "sqs.account"),
		Endpoint: config.GetString(ctx, "sqs.endpoint"),
		Region:   config.GetString(ctx, "aws.region"),
	}

	queueName := "CreateBulkOrderQueue"
	queue, err := sqs.NewStandardQueue(ctx, queueName, sqsCfg)
	if err != nil {
		log.Errorf(i18n.Translate(ctx, "NewBulkOrderPublisher: NewStandardQueue error: %v"), err)
		return nil, err
	}

	log.Infof(i18n.Translate(ctx, "SQS queue publisher ready"))
	return sqs.NewPublisher(queue), nil
}
