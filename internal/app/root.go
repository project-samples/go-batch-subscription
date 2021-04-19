package app

import (
	"github.com/common-go/health"
	"github.com/common-go/kafka"
	"github.com/common-go/log"
	"github.com/common-go/mongo"
	"github.com/common-go/mq"
)

type Root struct {
	Server            health.ServerConfig  `mapstructure:"server"`
	Log               log.Config           `mapstructure:"log"`
	Mongo             mongo.MongoConfig    `mapstructure:"mongo"`
	BatchWorkerConfig mq.BatchWorkerConfig `mapstructure:"batch_worker"`
	KafkaReader       kafka.ReaderConfig   `mapstructure:"kafka_reader"`
	KafkaWriter       *kafka.WriterConfig  `mapstructure:"kafka_writer"`
}
