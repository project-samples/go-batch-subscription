package app

import (
	"context"
	"github.com/core-go/mongo/geo"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/core-go/health"
	mgo "github.com/core-go/health/mongo"
	"github.com/core-go/mq"
	"github.com/core-go/mq/kafka"
	"github.com/core-go/mq/log"
	v "github.com/core-go/mq/validator"

	"go-service/pkg/batch"
)

type ApplicationContext struct {
	HealthHandler *health.Handler
	Run           func(ctx context.Context)
	Receive       func(context.Context, func(context.Context, []byte, map[string]string))
	Handle        func(context.Context, []byte, map[string]string)
}

func NewApp(ctx context.Context, root Root) (*ApplicationContext, error) {
	log.Initialize(root.Log)
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(root.Mongo.Uri))
	if err != nil {
		return nil, err
	}
	db := client.Database(root.Mongo.Database)
	if err != nil {
		return nil, err
	}

	logError := log.ErrorMsg
	var logInfo func(context.Context, string)
	if log.IsInfoEnable() {
		logInfo = log.InfoMsg
	}

	receiver, er2 := kafka.NewReaderByConfig(root.Reader, logError, true)
	if er2 != nil {
		log.Error(ctx, "Cannot create a new receiver. Error: "+er2.Error())
		return nil, er2
	}
	mongoChecker := mgo.NewHealthChecker(client)
	receiverChecker := kafka.NewKafkaHealthChecker(root.Reader.Brokers, "kafka_reader")

	mapper := geo.NewMapper[User]()
	batchWriter := batch.NewBatchInserter[User](db, "user", mapper.ModelToDb)
	batchHandler := mq.NewBatchHandler[User](batchWriter.Write)
	validator, err := v.NewValidator[*User]()
	if err != nil {
		return nil, err
	}
	errorHandler := mq.NewErrorHandler[*User](logError)

	sender, er3 := kafka.NewWriterByConfig(*root.Writer)
	if er3 != nil {
		log.Error(ctx, "Cannot new a new sender. Error: "+er3.Error())
		return nil, er3
	}
	senderChecker := kafka.NewKafkaHealthChecker(root.Writer.Brokers, "kafka_writer")

	batchWorker := mq.NewBatchWorkerByConfig[User](root.Batch, batchHandler.Handle, validator.Validate, errorHandler.RejectWithMap, errorHandler.HandleErrorWithMap, sender.Publish, logError, logInfo)
	batchWorker.LogDebug = logInfo

	healthHandler := health.NewHandler(mongoChecker, receiverChecker, senderChecker)

	return &ApplicationContext{
		HealthHandler: healthHandler,
		Run:           batchWorker.Run,
		Receive:       receiver.Read,
		Handle:        batchWorker.Handle,
	}, nil
}
