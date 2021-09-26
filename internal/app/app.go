package app

import (
	"context"
	"reflect"
	"strings"

	"github.com/core-go/health"
	mgo "github.com/core-go/mongo"
	"github.com/core-go/mq"
	"github.com/core-go/mq/log"
	"github.com/core-go/mq/sarama"
	v "github.com/core-go/mq/validator"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

type ApplicationContext struct {
	HealthHandler *health.Handler
	BatchWorker   mq.BatchWorker
	Receive       func(ctx context.Context, handle func(context.Context, *mq.Message, error) error)
	Subscription  *mq.Subscription
}

func NewApp(ctx context.Context, root Root) (*ApplicationContext, error) {
	log.Initialize(root.Log)
	db, er1 := mgo.Setup(ctx, root.Mongo)
	if er1 != nil {
		log.Error(ctx, "Cannot connect to MongoDB. Error: "+er1.Error())
		return nil, er1
	}

	logError := log.ErrorMsg
	var logInfo func(context.Context, string)
	if log.IsInfoEnable() {
		logInfo = log.InfoMsg
	}

	receiver, er2 := kafka.NewReaderByConfig(root.Reader, true)
	if er2 != nil {
		log.Error(ctx, "Cannot create a new receiver. Error: "+er2.Error())
		return nil, er2
	}

	userType := reflect.TypeOf(User{})
	batchWriter := mgo.NewBatchInserter(db, "user")
	batchHandler := mq.NewBatchHandler(batchWriter.Write, userType, logError, logInfo)

	mongoChecker := mgo.NewHealthChecker(db)
	receiverChecker := kafka.NewKafkaHealthChecker(root.Reader.Brokers, "kafka_reader")
	var healthHandler *health.Handler
	var batchWorker mq.BatchWorker

	if root.Writer != nil {
		sender, er3 := kafka.NewWriterByConfig(*root.Writer, Generate)
		if er3 != nil {
			log.Error(ctx, "Cannot new a new sender. Error: "+er3.Error())
			return nil, er3
		}
		retryService := mq.NewRetryService(sender.Write, logError, logInfo)
		batchWorker = mq.NewDefaultBatchWorker(root.BatchWorkerConfig, batchHandler.Handle, retryService.Retry, logError, logInfo)
		senderChecker := kafka.NewKafkaHealthChecker(root.Writer.Brokers, "kafka_writer")
		healthHandler = health.NewHandler(mongoChecker, receiverChecker, senderChecker)
	} else {
		batchWorker = mq.NewDefaultBatchWorker(root.BatchWorkerConfig, batchHandler.Handle, nil, logError, logInfo)
		healthHandler = health.NewHandler(mongoChecker, receiverChecker)
	}
	checker := v.NewErrorChecker(NewUserValidator().Validate)
	validator := mq.NewValidator(userType, checker.Check)
	subscription := mq.NewSubscription(batchWorker.Handle, validator.Validate, logError, logInfo)

	return &ApplicationContext{
		HealthHandler: healthHandler,
		BatchWorker:   batchWorker,
		Receive:       receiver.Read,
		Subscription:  subscription,
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
func Generate() string {
	return strings.Replace(uuid.New().String(), "-", "", -1)
}
