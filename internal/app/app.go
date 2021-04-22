package app

import (
	"context"
	"github.com/common-go/health"
	"github.com/common-go/log"
	"github.com/common-go/mongo"
	"github.com/common-go/mq"
	"github.com/common-go/rabbitmq"
	v "github.com/common-go/validator"
	"github.com/go-playground/validator/v10"
	"github.com/sirupsen/logrus"
	"reflect"
)

type ApplicationContext struct {
	Consumer        mq.Consumer
	ConsumerHandler mq.ConsumerHandler
	BatchWorker     mq.BatchWorker
	HealthHandler   *health.HealthHandler
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
	consumer, err := rabbitmq.NewConsumerByConfig(root.RabbitConsumer, true, true)
	if err != nil {
		return nil, err
	}

	userTypeOf := reflect.TypeOf(User{})
	bulkWriter := mongo.NewBatchInserter(mongoDb, "users")
	batchHandler := mq.NewBatchHandler(userTypeOf, bulkWriter.Write, logError, logInfo)

	mongoChecker := mongo.NewHealthChecker(mongoDb)
	consumerChecker := rabbitmq.NewRabbitMQHealthChecker(root.RabbitConsumer.Url)
	var checkers []health.HealthChecker
	var batchWorker mq.BatchWorker
	if root.RabbitProducer != nil {
		producer, er3 := rabbitmq.NewProducerByConfig(*root.RabbitProducer)
		if er3 != nil {
			log.Error(ctx, "Cannot create a new producer. Error : ", er3.Error())
		}

		if err != nil {
			log.Error(ctx, "Cannot declare exchange producer. Error : ", err)
		}

		retryService := mq.NewMqRetryService(producer.Produce, logError, logInfo)
		batchWorker = mq.NewDefaultBatchWorker(root.BatchWorkerConfig, batchHandler.Handle, retryService.Retry, logError, logInfo)
		producerChecker := rabbitmq.NewRabbitMQHealthChecker(root.RabbitProducer.Url)
		checkers = []health.HealthChecker{mongoChecker, consumerChecker, producerChecker}
	} else {
		batchWorker = mq.NewDefaultBatchWorker(root.BatchWorkerConfig, batchHandler.Handle, nil, logError, logInfo)
		checkers = []health.HealthChecker{mongoChecker, consumerChecker}

	}
	validator := mq.NewValidator(userTypeOf, NewUserValidator().Validate)
	consumerHandler := mq.NewBatchConsumerHandler(batchWorker.Consume, validator.Validate, logError, logInfo)
	handler := health.NewHealthHandler(checkers)
	return &ApplicationContext{
		Consumer:        consumer,
		ConsumerHandler: consumerHandler,
		BatchWorker:     batchWorker,
		HealthHandler:   handler,
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
