
package app

import (
	"context"
	"github.com/core-go/cassandra"
	"github.com/core-go/health"
	cas "github.com/core-go/health/cassandra"
	"github.com/core-go/mq"
	"github.com/core-go/mq/kafka"
	"github.com/core-go/mq/log"
	"github.com/core-go/mq/validator"
	val "github.com/go-playground/validator/v10"
	"github.com/gocql/gocql"
	"reflect"
)

type ApplicationContext struct {
	HealthHandler *health.Handler
	BatchWorker   mq.BatchWorker
	Receive       func(ctx context.Context, handle func(context.Context, *mq.Message, error) error)
	Subscription  *mq.Subscription
}

func NewApp(ctx context.Context, root Root) (*ApplicationContext, error) {
	log.Initialize(root.Log)
	cluster := gocql.NewCluster(root.Cassandra.Uri)
	cluster.Authenticator = gocql.PasswordAuthenticator{
		Username: root.Cassandra.Username,
		Password: root.Cassandra.Password,
	}
	session, er1 := cluster.CreateSession()
	if er1 != nil {
		log.Error(ctx, "Cannot connect to Cassandra, Error: " + er1.Error())
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
	batchWriter := 	cassandra.NewBatchWriter(session, "user.user", userType)
	batchHandler := mq.NewBatchHandler(userType, batchWriter.Write, logError, logInfo)

	cassandraChecker := cas.NewHealthChecker(cluster)
	receiverChecker := kafka.NewKafkaHealthChecker(root.Reader.Brokers, "kafka_reader")
	var healthHandler *health.Handler
	var batchWorker mq.BatchWorker

	if root.Writer != nil {
		sender, er3 := kafka.NewWriterByConfig(*root.Writer)
		if er3 != nil {
			log.Error(ctx, "Cannot new a new sender. Error: "+er3.Error())
			return nil, er3
		}
		retryService := mq.NewRetryService(sender.Write, logError, logInfo)
		batchWorker = mq.NewDefaultBatchWorker(root.BatchWorkerConfig, batchHandler.Handle, retryService.Retry, logError, logInfo)
		senderChecker := kafka.NewKafkaHealthChecker(root.Writer.Brokers, "kafka_writer")
		healthHandler = health.NewHandler(cassandraChecker, receiverChecker, senderChecker)
	} else {
		batchWorker = mq.NewDefaultBatchWorker(root.BatchWorkerConfig, batchHandler.Handle, nil, logError, logInfo)
		healthHandler = health.NewHandler(cassandraChecker, receiverChecker)
	}
	checker := validator.NewErrorChecker(NewUserValidator().Validate)
	validator := mq.NewValidator(userType, checker.Check)
	subscription := mq.NewSubscription(batchWorker.Handle, validator.Validate, logError, logInfo)

	return &ApplicationContext{
		HealthHandler: healthHandler,
		BatchWorker:   batchWorker,
		Receive:       receiver.Read,
		Subscription:  subscription,
	}, nil
}

func NewUserValidator() validator.Validator {
	val := validator.NewDefaultValidator()
	val.CustomValidateList = append(val.CustomValidateList, validator.CustomValidate{Fn: CheckActive, Tag: "active"})
	return val
}
func CheckActive(fl val.FieldLevel) bool {
	return fl.Field().Bool()
}
