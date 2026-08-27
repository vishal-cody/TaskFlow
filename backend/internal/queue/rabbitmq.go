// package queue manages rabbitmq connectivity, topology, and message publishing.
package queue

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// rabbitmq exchange, queue, and routing key constants.
const (
	ExchangeName = "jobs"
	ExchangeKind = "direct"

	QueueProcess = "jobs.process"
	QueueRetry   = "jobs.retry"
	QueueDLQ     = "jobs.dlq"

	RoutingKeyCreated = "job.created"
	RoutingKeyRetry   = "job.retry"

	// retry queue messages are re-routed to the process queue after ttl expires.
	RetryTTLMs = 5000 // 5 seconds default retry delay
)

// connection wraps an amqp connection with reconnect-friendly accessors.
type Connection struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	url     string
	logger  *slog.Logger
	mu      sync.Mutex
}

// newconnection dials rabbitmq and sets up the exchange/queue topology.
func NewConnection(ctx context.Context, url string, logger *slog.Logger) (*Connection, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("dialing RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("opening channel: %w", err)
	}

	c := &Connection{
		conn:    conn,
		channel: ch,
		url:     url,
		logger:  logger,
	}

	if err := c.declareTopology(); err != nil {
		c.Close()
		return nil, fmt.Errorf("declaring topology: %w", err)
	}

	logger.Info("connected to RabbitMQ", "url", sanitizeURL(url))
	return c, nil
}

// declaretopology creates the exchange, queues, and bindings.
//
// topology:
//
//	exchange "jobs" (direct)
//	  ├── routing_key "job.created" → queue "jobs.process"
//	  └── routing_key "job.retry"   → queue "jobs.retry"
//	                                      │ ttl expires
//	                                      ▼
//	                                  queue "jobs.process" (via dlx)
//
//	queue "jobs.process"
//	  └── on reject (no requeue) → queue "jobs.dlq" (via dlx)
func (c *Connection) declareTopology() error {
	ch := c.channel

	// declare the main exchange.
	if err := ch.ExchangeDeclare(ExchangeName, ExchangeKind, true, false, false, false, nil); err != nil {
		return fmt.Errorf("declaring exchange %q: %w", ExchangeName, err)
	}

	// dead letter queue — terminal destination for permanently failed jobs.
	if _, err := ch.QueueDeclare(QueueDLQ, true, false, false, false, nil); err != nil {
		return fmt.Errorf("declaring DLQ: %w", err)
	}

	// main processing queue.
	// messages rejected from this queue go to the dlq.
	if _, err := ch.QueueDeclare(QueueProcess, true, false, false, false, amqp.Table{
		"x-dead-letter-exchange":    "",       // default exchange
		"x-dead-letter-routing-key": QueueDLQ, // route to DLQ
	}); err != nil {
		return fmt.Errorf("declaring process queue: %w", err)
	}

	// retry queue — messages sit here for retryttlms then are re-delivered
	// to the process queue via dlx.
	if _, err := ch.QueueDeclare(QueueRetry, true, false, false, false, amqp.Table{
		"x-dead-letter-exchange":    "",           // default exchange
		"x-dead-letter-routing-key": QueueProcess, // re-route to process queue
		"x-message-ttl":             int32(RetryTTLMs),
	}); err != nil {
		return fmt.Errorf("declaring retry queue: %w", err)
	}

	// bind queues to the exchange.
	if err := ch.QueueBind(QueueProcess, RoutingKeyCreated, ExchangeName, false, nil); err != nil {
		return fmt.Errorf("binding process queue: %w", err)
	}
	if err := ch.QueueBind(QueueRetry, RoutingKeyRetry, ExchangeName, false, nil); err != nil {
		return fmt.Errorf("binding retry queue: %w", err)
	}

	return nil
}

// publish sends a message to the given exchange with the given routing key.
// uses publisher confirms implicitly via mandatory=false (fire-and-forget at amqp level;
// the outbox pattern provides the durability guarantee).
func (c *Connection) Publish(ctx context.Context, routingKey string, body []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.channel.PublishWithContext(ctx,
		ExchangeName,
		routingKey,
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Timestamp:    time.Now(),
			Body:         body,
		},
	)
}

// channel returns the underlying amqp channel for consumer use.
func (c *Connection) Channel() *amqp.Channel {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.channel
}

// close shuts down the channel and connection.
func (c *Connection) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.channel != nil {
		c.channel.Close()
	}
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// isclosed returns true if the connection is closed.
func (c *Connection) IsClosed() bool {
	return c.conn.IsClosed()
}

// sanitizeurl removes credentials from the url for logging.
func sanitizeURL(url string) string {
	// simple approach: just show the host.
	if len(url) > 30 {
		return url[:10] + "***"
	}
	return "amqp://***"
}
