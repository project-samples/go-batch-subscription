package app

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/core-go/health"
	mgo "github.com/core-go/health/mongo"
	"github.com/core-go/kafka"
	"github.com/core-go/mongo/batch"
	"github.com/core-go/mongo/geo"
	"github.com/core-go/mq"
	v "github.com/core-go/mq/validator"
	"github.com/core-go/mq/zap"
)

type ApplicationContext struct {
	HealthHandler *health.Handler
	Run           func(ctx context.Context)
	Read          func(context.Context, func(context.Context, []byte, map[string]string))
	Handle        func(context.Context, []byte, map[string]string)
}

func NewApp(ctx context.Context, cfg Config) (*ApplicationContext, error) {
	log.Initialize(cfg.Log)
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.Mongo.Uri))
	if err != nil {
		return nil, err
	}
	db := client.Database(cfg.Mongo.Database)
	if err != nil {
		return nil, err
	}

	logError := log.ErrorMsg
	var logInfo func(context.Context, string)
	if log.IsInfoEnable() {
		logInfo = log.InfoMsg
	}
	var logDebug func(context.Context, string)
	if log.IsDebugEnable() {
		logDebug = log.DebugMsg
	}

	reader, er2 := kafka.NewReaderByConfig(cfg.Reader, logError, true)
	if er2 != nil {
		log.Error(ctx, "Cannot create a new reader. Error: "+er2.Error())
		return nil, er2
	}
	mongoChecker := mgo.NewHealthChecker(client)
	receiverChecker := kafka.NewKafkaHealthChecker(cfg.Reader.Brokers, "kafka_reader")

	mapper := geo.NewMapper[User]()
	batchWriter := batch.NewBatchWriter[User](db, "user", mapper.ModelToDb)
	batchHandler := mq.NewBatchHandler[User](batchWriter.Write)
	validator, err := v.NewValidator[*User]()
	if err != nil {
		return nil, err
	}
	errorHandler := mq.NewErrorHandler[*User](logError)

	sender, er3 := kafka.NewWriterByConfig(*cfg.Writer)
	if er3 != nil {
		log.Error(ctx, "Cannot new a new sender. Error: "+er3.Error())
		return nil, er3
	}
	senderChecker := kafka.NewKafkaHealthChecker(cfg.Writer.Brokers, "kafka_writer")

	batchWorker := mq.NewBatchWorkerByConfig[User](cfg.Batch, batchHandler.Handle, validator.Validate, errorHandler.RejectWithMap, errorHandler.HandleErrorWithMap, sender.Write, logError, logInfo, logDebug)
	batchWorker.LogDebug = logInfo

	healthHandler := health.NewHandler(mongoChecker, receiverChecker, senderChecker)

	return &ApplicationContext{
		HealthHandler: healthHandler,
		Run:           batchWorker.Run,
		Read:          reader.Read,
		Handle:        batchWorker.Handle,
	}, nil
}
