package app

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	// "github.com/core-go/firestore"
	//"github.com/core-go/firestore/batch"
	"github.com/core-go/health"
	mh "github.com/core-go/health/mongo"
	"github.com/core-go/mongo/batch"
	"github.com/core-go/mq"
	"github.com/core-go/mq/pubsub"
	v "github.com/core-go/mq/validator"
	"github.com/core-go/mq/zap"
)

type ApplicationContext struct {
	Health  *health.Handler
	Run     func(ctx context.Context)
	Receive func(context.Context, func(context.Context, []byte, map[string]string))
	Handle  func(context.Context, []byte, map[string]string)
}

func NewApp(ctx context.Context, cfg Config) (*ApplicationContext, error) {
	log.Initialize(cfg.Log)
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.Mongo.Uri))
	db := client.Database(cfg.Mongo.Database)
	if err != nil {
		return nil, err
	}
	/*
		client, er1 := firestore.Connect(ctx, []byte(cfg.Firestore.Credentials))
		if er1 != nil {
			return nil, er1
		}*/

	logError := log.ErrorMsg
	var logInfo func(context.Context, string)
	if log.IsInfoEnable() {
		logInfo = log.InfoMsg
	}

	receiver, er2 := pubsub.NewSubscriberByConfig(ctx, cfg.Sub, logError, true)
	if er2 != nil {
		log.Error(ctx, "Cannot create a new receiver. Error: "+er2.Error())
		return nil, er2
	}

	batchWriter := batch.NewBatchInserter[User](db, "user")
	batchHandler := mq.NewBatchHandler[User](batchWriter.Write)
	validator, err := v.NewValidator[*User]()
	if err != nil {
		return nil, err
	}
	errorHandler := mq.NewErrorHandler[*User](logError)

	mongoChecker := mh.NewHealthChecker(client)
	receiverChecker := pubsub.NewSubHealthChecker("pubsub_subscriber", receiver.Client, cfg.Sub.SubscriptionId)

	sender, er3 := pubsub.NewPublisherByConfig(ctx, *cfg.Pub)
	if er3 != nil {
		log.Error(ctx, "Cannot new a new sender. Error: "+er3.Error())
		return nil, er3
	}
	batchWorker := mq.NewBatchWorkerByConfig[User](cfg.Batch, batchHandler.Handle, validator.Validate, errorHandler.RejectWithMap, errorHandler.HandleErrorWithMap, sender.Publish, logError, logInfo, log.DebugMsg)
	batchWorker.LogDebug = logInfo
	senderChecker := pubsub.NewPubHealthChecker("pubsub_publisher", sender.Client, cfg.Pub.TopicId)
	healthHandler := health.NewHandler(mongoChecker, receiverChecker, senderChecker)

	return &ApplicationContext{
		Health:  healthHandler,
		Run:     batchWorker.Run,
		Receive: receiver.Subscribe,
		Handle:  batchWorker.Handle,
	}, nil
}
