package broker

import (
	"strconv"

	"github.com/IBM/sarama"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/config"
	"go.uber.org/thriftrw/ptr"
)

func CreateTopicIfNotExists(kafkaCfg *config.KafkaConfig, cfg *sarama.Config) error {
	admin, err := sarama.NewClusterAdmin(kafkaCfg.Brokers, cfg)
	if err != nil {
		return err
	}
	defer admin.Close()

	topics, err := admin.ListTopics()
	if err != nil {
		return err
	}

	if _, exists := topics[kafkaCfg.Raw.Topic]; !exists {
		err = admin.CreateTopic(
			kafkaCfg.Raw.Topic,
			&sarama.TopicDetail{
				NumPartitions:     kafkaCfg.NumPartitions,
				ReplicationFactor: int16(len(kafkaCfg.Brokers)),
				ConfigEntries: map[string]*string{
					"cleanup.policy":      ptr.String("delete"),
					"retention.ms":        ptr.String(strconv.Itoa(kafkaCfg.RetentionMs)),
					"min.insync.replicas": ptr.String(strconv.Itoa(kafkaCfg.MinInsyncReplicas)),
				},
			},
			false)
		if err != nil {
			return err
		}
	}

	return nil
}
