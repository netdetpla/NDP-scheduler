package main

import (
	"encoding/json"
	"github.com/Shopify/sarama"
)

type kafkaResultJSON struct {
	TaskID string `json:"task_id"`
	MD5 string `json:"md5"`
	ResultLine string `json:"resultline"`
}

func extractResult(resultLine string, taskID string, table string)  {
	log.Debugf("Consumed message %s", resultLine)
	scanOpt <- dbOpt{"result", []string{resultLine, taskID, table}}
}

var resultTables = map[string]string {
	"scanWebTaskFile": "scanweb",
	"scanServiceTaskFile": "scanservice",
	"scanDnsTaskFile": "scandns",
	"bugTaskFile": "info_shell",
	"scanVulTaskFile": "scanvul",
	"ecdsystemTaskFile": "ecdsystem",
	"domainTaskFile": "dnssecure",
	"nsTaskFile": "dnsns",
}

func generateConsumer(topic string) {
	consumer, err := sarama.NewConsumer([]string{
		"192.168.120.10:9092", "192.168.120.11:9092", "192.168.120.12:9092"}, nil)
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
		go extractResult(msgJSON.ResultLine, msgJSON.TaskID, resultTables[msg.Topic])
	}
}
func consumersManager() {
	log.Notice("consumers manager started.")
	for topic := range resultTables {
		go generateConsumer(topic)
	}
}
