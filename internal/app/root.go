package app

import (
	"github.com/core-go/health/server"
	"github.com/core-go/mq"
	"github.com/core-go/mq/log"
	"github.com/core-go/mq/sarama"
	"github.com/core-go/sql"
)

type Root struct {
	Server            server.ServerConf    `mapstructure:"server"`
	Log               log.Config           `mapstructure:"log"`
	Sql               sql.Config           `mapstructure:"sql"`
	BatchWorkerConfig mq.BatchWorkerConfig `mapstructure:"batch_worker"`
	Reader            kafka.ReaderConfig   `mapstructure:"reader"`
	Writer            *kafka.WriterConfig  `mapstructure:"writer"`
}
