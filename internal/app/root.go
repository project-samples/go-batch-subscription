package app

import (
	"github.com/common-go/health"
	"github.com/common-go/log"
	"github.com/common-go/mongo"
	"github.com/common-go/mq"
	"github.com/common-go/rabbitmq"
)

type Root struct {
	Server            health.ServerConfig   `mapstructure:"server"`
	Log               log.Config            `mapstructure:"log"`
	Mongo             mongo.MongoConfig     `mapstructure:"mongo"`
	BatchWorkerConfig mq.BatchWorkerConfig  `mapstructure:"batch_worker"`
	RabbitConsumer    rabbitmq.ConsumerConfig  `mapstructure:"rabbitmq_consumer"`
	RabbitProducer    *rabbitmq.ProducerConfig `mapstructure:"rabbitmq_producer"`
}
