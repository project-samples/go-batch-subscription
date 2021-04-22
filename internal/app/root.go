package app

import (
	"github.com/common-go/health"
	"github.com/common-go/kafka"
	"github.com/common-go/log"
	"github.com/common-go/mq"
	s "github.com/common-go/sql"
	_ "github.com/go-sql-driver/mysql"
)

type Root struct {
	Server            health.ServerConfig  `mapstructure:"server"`
	Log               log.Config           `mapstructure:"log"`
	Mysql		      s.DatabaseConfig	   `mapstructure:"mysql"`
	BatchWorkerConfig mq.BatchWorkerConfig `mapstructure:"batch_worker"`
	KafkaReader       kafka.ReaderConfig   `mapstructure:"kafka_reader"`
	KafkaWriter       *kafka.WriterConfig  `mapstructure:"kafka_writer"`
}
