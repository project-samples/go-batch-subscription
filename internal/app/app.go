package app

import (
	"context"
	"reflect"

	"github.com/common-go/health"
	"github.com/common-go/kafka"
	"github.com/common-go/log"
	"github.com/common-go/mongo"
	"github.com/common-go/mq"
	v "github.com/common-go/validator"
	"github.com/go-playground/validator/v10"
	"github.com/sirupsen/logrus"
)

type ApplicationContext struct {
	HealthHandler   *health.HealthHandler
	Consume         func(ctx context.Context, handle func(context.Context, *mq.Message, error) error)
	ConsumerHandler mq.ConsumerHandler
	BatchWorker     mq.BatchWorker
}

func NewApp(ctx context.Context, root Root) (*ApplicationContext, error) {
	log.Initialize(root.Log)
	db, er1 := mongo.SetupMongo(ctx, root.Mongo)
	if er1 != nil {
		log.Error(ctx, "Cannot connect to MongoDB. Error: "+er1.Error())
		return nil, er1
	}

	logError := log.ErrorMsg
	var logInfo func(context.Context, string)
	if logrus.IsLevelEnabled(logrus.InfoLevel) {
		logInfo = log.InfoMsg
	}

	consumer, er2 := kafka.NewReaderByConfig(root.KafkaReader, true)
	if er2 != nil {
		log.Error(ctx, "Cannot create a new consumer. Error: "+er2.Error())
		return nil, er2
	}

	userType := reflect.TypeOf(User{})
	writer := mongo.NewBatchInserter(db, "users")
	batchHandler := mq.NewBatchHandler(userType, writer.Write, logError, logInfo)

	mongoChecker := mongo.NewHealthChecker(db)
	consumerChecker := kafka.NewKafkaHealthChecker(root.KafkaReader.Brokers, "kafka_reader")
	var checkers []health.HealthChecker
	var batchWorker mq.BatchWorker

	if root.KafkaWriter != nil {
		producer, er3 := kafka.NewWriterByConfig(*root.KafkaWriter)
		if er3 != nil {
			log.Error(ctx, "Cannot new a new producer. Error: "+er3.Error())
			return nil, er3
		}
		retryService := mq.NewMqRetryService(producer.Write, logError, logInfo)
		batchWorker = mq.NewDefaultBatchWorker(root.BatchWorkerConfig, batchHandler.Handle, retryService.Retry, logError, logInfo)
		producerChecker := kafka.NewKafkaHealthChecker(root.KafkaWriter.Brokers, "kafka_writer")
		checkers = []health.HealthChecker{mongoChecker, consumerChecker, producerChecker}
	} else {
		batchWorker = mq.NewDefaultBatchWorker(root.BatchWorkerConfig, batchHandler.Handle, nil, logError, logInfo)
		checkers = []health.HealthChecker{mongoChecker, consumerChecker}
	}
	validator := mq.NewValidator(userType, NewUserValidator().Validate, logError)
	consumerHandler := mq.NewBatchConsumerHandler(batchWorker.Consume, validator.Validate, logError, logInfo)

	handler := health.NewHealthHandler(checkers)
	return &ApplicationContext{
		HealthHandler:   handler,
		Consume:         consumer.Read,
		ConsumerHandler: consumerHandler,
		BatchWorker:     batchWorker,
	}, nil
}

func NewUserValidator() v.Validator {
	validator := v.NewDefaultValidator()
	validator.CustomValidateList = append(validator.CustomValidateList, v.CustomValidate{Fn: CheckActive, Tag: "active"})
	return validator
}
func CheckActive(fl validator.FieldLevel) bool {
	return fl.Field().Bool()
}
