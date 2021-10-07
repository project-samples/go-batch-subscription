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
	Cassandra   	  CassandraConfig      `mapstructure:"cassandra"`
	BatchWorkerConfig mq.BatchWorkerConfig `mapstructure:"batch_worker"`
	Reader            kafka.ReaderConfig   `mapstructure:"reader"`
	Writer            *kafka.WriterConfig  `mapstructure:"writer"`
}

type CassandraConfig struct {
	Uri      string `mapstructure:"uri" json:"uri,omitempty" gorm:"column:uri" bson:"uri,omitempty" dynamodbav:"uri,omitempty" firestore:"uri,omitempty"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}
