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
	"github.com/sirupsen/logrus"
	"gopkg.in/go-playground/validator.v9"
)

type ApplicationContext struct {
	Consumer       mq.Consumer
	ConsumerCaller mq.ConsumerCaller
	BatchWorker    mq.BatchWorker
	HealthHandler  *health.HealthHandler
}

func NewApp(ctx context.Context, root Root) (*ApplicationContext, error) {
	log.Initialize(root.Log)
	mongoDb, er1 := mongo.SetupMongo(ctx, root.Mongo)
	if er1 != nil {
		log.Error(ctx, "Cannot connect to MongoDB: Error: "+er1.Error())
		return nil, er1
	}

	logError := log.ErrorMsg
	var logInfo func(context.Context, string)
	if logrus.IsLevelEnabled(logrus.InfoLevel) {
		logInfo = log.InfoMsg
	}

	consumer, er2 := kafka.NewConsumerByConfig(root.KafkaConsumer, true)
	if er2 != nil {
		log.Error(ctx, "Can't new consumer: Error: " + er2.Error())
		return nil, er2
	}
	producer, er3 := kafka.NewProducerByConfig(root.KafkaProducer, true)
	if er3 != nil {
		log.Error(ctx, "Can't new producer: Error: " + er3.Error())
		return nil, er3
	}

	userTypeOf := reflect.TypeOf(User{})
	bulkWriter := mongo.NewMongoBatchInserter(mongoDb, "users")
	batchHandler := mq.NewBatchHandler(userTypeOf, bulkWriter, logError, logInfo)

	retryService := mq.NewMqRetryService(producer, logError, logInfo)
	batchWorker := mq.NewDefaultBatchWorker(root.BatchWorkerConfig, batchHandler, retryService, logError, logInfo)

	validator := mq.NewValidator(userTypeOf, NewUserValidator(), logError)
	consumerCaller := mq.NewBatchConsumerCaller(batchWorker, validator, logError, logInfo)

	mongoChecker := mongo.NewHealthChecker(mongoDb)
	consumerChecker := kafka.NewKafkaHealthChecker(root.KafkaConsumer.Brokers)
	checkers := []health.HealthChecker{mongoChecker, consumerChecker}
	handler := health.NewHealthHandler(checkers)
	return &ApplicationContext{
		Consumer:       consumer,
		ConsumerCaller: consumerCaller,
		BatchWorker:    batchWorker,
		HealthHandler:  handler,
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
