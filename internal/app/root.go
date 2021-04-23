package app

import (
	"github.com/common-go/amq"
	"github.com/common-go/health"
	"github.com/common-go/log"
	"github.com/common-go/mongo"
	"github.com/common-go/mq"
)

type Root struct {
	Server            health.ServerConfig  `mapstructure:"server"`
	Log               log.Config           `mapstructure:"log"`
	Mongo             mongo.MongoConfig    `mapstructure:"mongo"`
	BatchWorkerConfig mq.BatchWorkerConfig `mapstructure:"batch_worker"`
	AmqConfig         amq.Config           `mapstructure:"amq_config"`
}
