package app

import (
	"github.com/core-go/health/server"
	"github.com/core-go/mongo"
	"github.com/core-go/mq"
	"github.com/core-go/mq/kafka"
	"github.com/core-go/mq/log"
)

type Root struct {
	Server            server.ServerConf    `mapstructure:"server"`
	Log               log.Config           `mapstructure:"log"`
	Mongo             mongo.MongoConfig    `mapstructure:"mongo"`
	BatchWorkerConfig mq.BatchWorkerConfig `mapstructure:"batch_worker"`
	Reader            kafka.ReaderConfig   `mapstructure:"reader"`
	Writer            *kafka.WriterConfig  `mapstructure:"writer"`
}
