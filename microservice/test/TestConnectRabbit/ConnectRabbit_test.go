package TestConnectRabbit

import (
	"fmt"
	ConnectionRabbitMQ "lib-rabbitmq"
	"testing"
	"github.com/stretchr/testify/assert"
)

var configMap =  make(map[string]string)
var connRabbit ConnectionRabbitMQ.ChannelMQ
var Queue = make(map[string]interface{})


func initConfig() {
	configMap["Login"] = "testUnit"
	configMap["Password"] = "testUnit"
	configMap["Address"] = "188.225.18.140"
	configMap["Port"] = "5672"
}

func TestConnectRabbitMQ(t *testing.T) {
	initConfig()
	connRabbit.ConnectionToRabbit(configMap["Login"], configMap["Password"], configMap["Address"], configMap["Port"])
	assert.Equal(t,connRabbit.ConnectionChannel.IsReady,true)
}

func TestConnectChannel(t *testing.T){
	connRabbit.ConnectQueue()
	assert.Equal(t,connRabbit.ChannelConn,true)
}

func TestCreateQueueRabbit(t *testing.T){
	_,err := connRabbit.QueueDeclareRabbit("test.Rabbit")
	if err != nil {
		t.Errorf("[X]- Failed to create Queue")
	} else {
	 	fmt.Println("[S]- Success to create Queue")	
	}
}

func TestConsumerQueue(t *testing.T){
	_,err := connRabbit.RabbitMQConsume("test.Rabbit")
	if err != nil {
		t.Errorf("[X]- Failed to consume Queue")
	} else {
	 	fmt.Println("[S]- Success to consume Queue")	
	}
}

func TestCloseConnectionRabbit(t *testing.T){
	connRabbit.CloseConnectRabbit()
	if connRabbit.ChannelConn != true && connRabbit.ConnectionChannel.IsReady != true {
		fmt.Println("[S]- Success to close Connection")	
	}else {
		t.Errorf("[X]- Failed to close Connection")
	}
}


