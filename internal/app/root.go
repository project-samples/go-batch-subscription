package app

import (
	"github.com/core-go/elasticsearch"
	"github.com/core-go/health/server"
	"github.com/core-go/mq"
	"github.com/core-go/mq/log"
	"github.com/core-go/mq/sarama"
)

type Root struct {
	Server            server.ServerConf    `mapstructure:"server"`
	Log               log.Config           `mapstructure:"log"`
	ElasticSearch     elasticsearch.Config `mapstructure:"elasticsearch"`
	BatchWorkerConfig mq.BatchWorkerConfig `mapstructure:"batch_worker"`
	Reader            kafka.ReaderConfig   `mapstructure:"reader"`
	Writer            *kafka.WriterConfig  `mapstructure:"writer"`
}
