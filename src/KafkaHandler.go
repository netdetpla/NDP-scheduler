package main

import (
	"github.com/Shopify/sarama"
	"encoding/json"
)

type kafkaResultJSON struct {
	TaskID string `json:"task_id"`
	MD5 string `json:"md5"`
	ResultLine string `json:"resultline"`
}

func scanWebResultExtractor(resultLine string, taskID string, table string) {
	log.Debugf("Consumed message %s", resultLine)
	scanOpt <- dbOpt{"result", []string{resultLine, taskID, table}}
}

func scanServiceResultExtractor(resultLine string, taskID string, table string) {
	log.Debugf("Consumed message %s", resultLine)
}

var resultTopics = map[string]func(s string, id string, table string){
	"scanWebTaskFile": scanWebResultExtractor,
	"scanServiceResultExtractor": scanServiceResultExtractor,
}

var resultTables = map[string]string {
	"scanWebTaskFile": "scanweb",
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

	for {
		msg := <-partitionConsumer.Messages()
		msgJSON := new(kafkaResultJSON)
		err = json.Unmarshal(msg.Value, msgJSON)
		go resultTopics[msg.Topic](msgJSON.ResultLine, msgJSON.TaskID, resultTables[msg.Topic])
	}
}
func consumersManager() {
	log.Notice("consumers manager started.")
	for topic := range resultTopics {
		go generateConsumer(topic)
	}
}
