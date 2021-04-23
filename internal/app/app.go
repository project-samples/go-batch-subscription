package app

import (
	"context"
	"github.com/common-go/amq"
	"github.com/common-go/mongo"
	"github.com/go-stomp/stomp"
	"reflect"

	"github.com/common-go/health"
	"github.com/common-go/log"
	"github.com/common-go/mq"
	v "github.com/common-go/validator"
	"github.com/go-playground/validator/v10"
	"github.com/sirupsen/logrus"
)

type ApplicationContext struct {
	HealthHandler   *health.HealthHandler
	Consumer        mq.Consumer
	ConsumerHandler mq.ConsumerHandler
	BatchWorker     mq.BatchWorker
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

	consumer, err := amq.NewConsumerByConfig(root.AmqConfig, stomp.AckAuto, true)
	if err != nil {
		return nil, err
	}

	userType := reflect.TypeOf(User{})
	writer := mongo.NewBatchInserter(mongoDb, "users")
	batchHandler := mq.NewBatchHandler(userType, writer.Write, logError, logInfo)

	mongoChecker := mongo.NewHealthChecker(mongoDb)
	consumerChecker := amq.NewAMQHealthChecker(consumer.Conn)
	var checkers []health.HealthChecker
	var batchWorker mq.BatchWorker
	batchWorker = mq.NewDefaultBatchWorker(root.BatchWorkerConfig, batchHandler.Handle, nil, logError, logInfo)
	checkers = []health.HealthChecker{mongoChecker, consumerChecker}

	validator := mq.NewValidator(userType, NewUserValidator().Validate, logError)
	consumerHandler := mq.NewBatchConsumerHandler(batchWorker.Consume, validator.Validate, logError, logInfo)

	handler := health.NewHealthHandler(checkers)
	return &ApplicationContext{
		HealthHandler:   handler,
		Consumer:        consumer,
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
