package ConnectionRabbitMQ

import(
	"fmt"
	"log"
	"amqp"
)
type ChannelMQ struct {
	Channel         *amqp.Channel
	ConnectionChannel ConnectionRabbit
	ChannelConn bool
}

type ConnectionRabbit struct {
	Connect  *amqp.Connection
	IsReady         bool
}

func (ch *ChannelMQ) ConnectionToRabbit(login string, password string, address string, port string) error {
	stringConnection:= fmt.Sprintf("amqp://%s:%s@%s:%s",login,password,address,port)
	conn, err := amqp.Dial(stringConnection)
	if err != nil {
		failOnError(err, "Failed to connect to RabbitMQ")
		return err
	}else {
		ch.ConnectionChannel.Connect = conn
	    ch.ConnectionChannel.IsReady = true
		return nil
	}	
}

func (ch *ChannelMQ) ConnectQueue(){
	channel, err := ch.ConnectionChannel.Connect.Channel()
	if err != nil {
		ch.ChannelConn = false
	}
	failOnError(err, "Failed to open a channel")
	ch.Channel = channel
	ch.ChannelConn = true
}

func (ch *ChannelMQ) RabbitMQConsume(nameQueue string) (<-chan amqp.Delivery, error){
	msgs, err := ch.Channel.Consume(
		nameQueue, // queue
		"",     // consumer
		true,   // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	  )
	failOnError(err, "Failed to register a consumer")
	return msgs,err
}

func (ch *ChannelMQ) CloseConnectRabbit() {
	ch.Channel.Close()
	ch.ConnectionChannel.Connect.Close()
	ch.ChannelConn = false
	ch.ConnectionChannel.IsReady = false
}

func (ch *ChannelMQ) QueueDeclareRabbit(name string) (amqp.Queue, error){
	queue,err := ch.Channel.QueueDeclare(name, true, false, false, false, nil)
	return queue,err
}

func failOnError(err error, msg string) {
	if err != nil {
	  log.Fatalf("%s: %s", msg, err)
	}
}
