package main

import (
	"github.com/Shopify/sarama"
	"os"
	"os/signal"
)

func scanWebResultExtractor(resultLine string) {
	log.Debugf("Consumed message %s", resultLine)
}

func scanServiceResultExtractor(resultLine string) {
	log.Debugf("Consumed message %s", resultLine)
}

var resultTopics = map[string]func(s string){
	"scanWebTaskFile": scanWebResultExtractor,
	"scanServiceResultExtractor": scanServiceResultExtractor,
}

func generateConsumer(topic string) {
	consumer, err := sarama.NewConsumer([]string{
		"192.168.226.11:9092", "192.168.226.12:9092", "192.168.226.13:9092"}, nil)
	if err != nil {
		log.Warning(err.Error())
	}

	defer func() {
		if err := consumer.Close(); err != nil {
			log.Warning(err.Error())
		}
	}()

	partitionConsumer, err := consumer.ConsumePartition(topic, 0, sarama.OffsetNewest)
	if err != nil {
		log.Warning(err.Error())
	}

	defer func() {
		if err := partitionConsumer.Close(); err != nil {
			log.Warning(err.Error())
		}
	}()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)

ConsumerLoop:
	for {
		select {
		case msg := <-partitionConsumer.Messages():
			go resultTopics[msg.Topic](string(msg.Value))
		case <-signals:
			break ConsumerLoop
		}
	}
}
func consumersManager() {
	for topic := range resultTopics {
		go generateConsumer(topic)
	}
}
