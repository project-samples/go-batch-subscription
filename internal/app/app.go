package app

import (
	"context"
	"reflect"

	"github.com/core-go/health"
	mgo "github.com/core-go/mongo"
	"github.com/core-go/mq"
	"github.com/core-go/mq/log"
	"github.com/core-go/mq/sqs"
	v "github.com/core-go/mq/validator"
	"github.com/go-playground/validator/v10"
)

type ApplicationContext struct {
	HealthHandler *health.HealthHandler
	BatchWorker   mq.BatchWorker
	Receive       func(ctx context.Context, handle func(context.Context, *mq.Message, error) error)
	Subscription  *mq.Subscription
}

func NewApp(ctx context.Context, root Root) (*ApplicationContext, error) {
	log.Initialize(root.Log)
	db, er0 := mgo.SetupMongo(ctx, root.Mongo)
	if er0 != nil {
		log.Error(ctx, "Cannot connect to MongoDB. Error: "+er0.Error())
		return nil, er0
	}

	logError := log.ErrorMsg
	var logInfo func(context.Context, string)
	if log.IsInfoEnable() {
		logInfo = log.InfoMsg
	}

	receiverClient, er2 := sqs.Connect(root.Receiver)
	if er2 != nil {
		log.Error(ctx, "Cannot create a new receiver. Error: "+er2.Error())
		return nil, er2
	}
	receiver := sqs.NewReceiver(receiverClient, root.Receiver.QueueName, true, 20, 1)

	userType := reflect.TypeOf(User{})
	batchWriter := mgo.NewBatchInserter(db, "users")
	batchHandler := mq.NewBatchHandler(userType, batchWriter.Write, logError, logInfo)

	mongoChecker := mgo.NewHealthChecker(db)
	receiverChecker := sqs.NewHealthChecker(receiverClient, root.Receiver.QueueName, "sqs_receiver")
	var healthHandler *health.HealthHandler
	var batchWorker mq.BatchWorker

	if root.Sender != nil {
		senderClient, er3 := sqs.Connect(*root.Sender)
		if er3 != nil {
			log.Error(ctx, "Cannot new a new sender. Error: "+er3.Error())
			return nil, er3
		}
		var delaySecond int64
		delaySecond = 1
		sender := sqs.NewSender(senderClient, root.Sender.QueueName, &delaySecond)

		retryService := mq.NewRetryService(sender.Send, logError, logInfo)
		batchWorker = mq.NewDefaultBatchWorker(root.BatchWorkerConfig, batchHandler.Handle, retryService.Retry, logError, logInfo)
		senderChecker := sqs.NewHealthChecker(senderClient, root.Sender.QueueName, "sqs_sender")
		healthHandler = health.NewHealthHandler(mongoChecker, receiverChecker, senderChecker)
	} else {
		batchWorker = mq.NewDefaultBatchWorker(root.BatchWorkerConfig, batchHandler.Handle, nil, logError, logInfo)
		healthHandler = health.NewHealthHandler(mongoChecker, receiverChecker)
	}
	checker := v.NewErrorChecker(NewUserValidator().Validate)
	validator := mq.NewValidator(userType, checker.Check)
	subscription := mq.NewSubscription(batchWorker.Handle, validator.Validate, logError, logInfo)

	return &ApplicationContext{
		HealthHandler: healthHandler,
		BatchWorker:   batchWorker,
		Receive:       receiver.Receive,
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
