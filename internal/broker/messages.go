package broker

import (
	"strconv"

	"github.com/IBM/sarama"
)

func CreateMessages(howMany int, topic string) []*sarama.ProducerMessage {
	messages := make([]*sarama.ProducerMessage, 0, howMany)

	for i := range howMany {
		messages = append(messages, CreateMessage(i, topic))
	}

	return messages
}

func CreateMessage(number int, topic string) *sarama.ProducerMessage {
	key := "odd"
	if number%2 == 0 {
		key = "even"
	}

	value := strconv.Itoa(number)

	return &sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.StringEncoder(value),
		Key:   sarama.StringEncoder(key),
	}
}
