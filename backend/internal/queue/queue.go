package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Task represents an async task to process.
type Task struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	RoomID    string                 `json:"room_id"`
	Data      map[string]interface{} `json:"data"`
	Priority  int                    `json:"priority"`
	CreatedAt time.Time              `json:"created_at"`
	Retries   int                    `json:"retries"`
	MaxRetry  int                    `json:"max_retry"`
}

// TaskResult represents the result of a task.
type TaskResult struct {
	TaskID    string                 `json:"task_id"`
	Success   bool                   `json:"success"`
	Result    map[string]interface{} `json:"result,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Duration  time.Duration          `json:"duration"`
	Timestamp time.Time              `json:"timestamp"`
}

// TaskHandler handles task processing.
type TaskHandler func(ctx context.Context, task Task) (map[string]interface{}, error)

// Queue manages RabbitMQ task queue.
type Queue struct {
	conn       *amqp.Connection
	channel    *amqp.Channel
	handlers   map[string]TaskHandler
	mu         sync.RWMutex
	logger     *slog.Logger
	queueName  string
	resultCh   chan TaskResult
	ctx        context.Context
	cancelFunc context.CancelFunc
}

// Config for the queue.
type Config struct {
	URL        string
	QueueName  string
	Prefetch   int
	Logger     *slog.Logger
	RetryDelay time.Duration
	MaxRetries int
}

// New creates a new task queue.
func New(cfg Config) (*Queue, error) {
	conn, err := amqp.Dial(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	if err := ch.Qos(cfg.Prefetch, 0, false); err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to set QoS: %w", err)
	}

	_, err = ch.QueueDeclare(
		cfg.QueueName,
		true,
		false,
		false,
		false,
		amqp.Table{"x-max-priority": 10},
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare queue: %w", err)
	}

	dlqName := cfg.QueueName + "_dlq"
	_, err = ch.QueueDeclare(dlqName, true, false, false, false, nil)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare DLQ: %w", err)
	}

	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Queue{
		conn:       conn,
		channel:    ch,
		handlers:   make(map[string]TaskHandler),
		logger:     logger,
		queueName:  cfg.QueueName,
		resultCh:   make(chan TaskResult, 100),
		ctx:        ctx,
		cancelFunc: cancel,
	}, nil
}

// RegisterHandler registers a handler for a task type.
func (q *Queue) RegisterHandler(taskType string, handler TaskHandler) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.handlers[taskType] = handler
}

// Publish publishes a task to the queue.
func (q *Queue) Publish(ctx context.Context, task Task) error {
	if task.CreatedAt.IsZero() {
		task.CreatedAt = time.Now()
	}
	if task.MaxRetry == 0 {
		task.MaxRetry = 3
	}

	body, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	return q.channel.PublishWithContext(
		ctx,
		"",
		q.queueName,
		false,
		false,
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "application/json",
			Body:         body,
			Priority:     uint8(task.Priority),
			MessageId:    task.ID,
			Timestamp:    task.CreatedAt,
		},
	)
}

// Start starts consuming tasks.
func (q *Queue) Start(ctx context.Context) error {
	msgs, err := q.channel.Consume(q.queueName, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("failed to start consuming: %w", err)
	}

	go q.processMessages(ctx, msgs)
	return nil
}

func (q *Queue) processMessages(ctx context.Context, msgs <-chan amqp.Delivery) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-q.ctx.Done():
			return
		case msg, ok := <-msgs:
			if !ok {
				return
			}
			q.processMessage(ctx, msg)
		}
	}
}

func (q *Queue) processMessage(ctx context.Context, msg amqp.Delivery) {
	var task Task
	if err := json.Unmarshal(msg.Body, &task); err != nil {
		q.logger.Error("Failed to unmarshal task", "error", err)
		msg.Nack(false, false)
		return
	}

	q.mu.RLock()
	handler, ok := q.handlers[task.Type]
	q.mu.RUnlock()

	if !ok {
		q.logger.Error("No handler for task type", "type", task.Type)
		msg.Nack(false, false)
		return
	}

	start := time.Now()
	result, err := handler(ctx, task)
	duration := time.Since(start)

	taskResult := TaskResult{
		TaskID:    task.ID,
		Timestamp: time.Now(),
		Duration:  duration,
	}

	if err != nil {
		taskResult.Success = false
		taskResult.Error = err.Error()

		if task.Retries < task.MaxRetry {
			task.Retries++
			if rerr := q.Publish(ctx, task); rerr != nil {
				q.logger.Error("Failed to requeue task", "error", rerr)
			}
		} else {
			dlqName := q.queueName + "_dlq"
			q.channel.PublishWithContext(ctx, "", dlqName, false, false, amqp.Publishing{
				ContentType: "application/json",
				Body:        msg.Body,
			})
		}
		msg.Nack(false, false)
	} else {
		taskResult.Success = true
		taskResult.Result = result
		msg.Ack(false)
	}

	select {
	case q.resultCh <- taskResult:
	default:
	}
}

// Results returns the result channel.
func (q *Queue) Results() <-chan TaskResult {
	return q.resultCh
}

// Close closes the queue connection.
func (q *Queue) Close() error {
	q.cancelFunc()
	if err := q.channel.Close(); err != nil {
		return err
	}
	return q.conn.Close()
}

// HealthCheck checks if the queue is healthy.
func (q *Queue) HealthCheck() error {
	if q.conn.IsClosed() {
		return fmt.Errorf("connection closed")
	}
	return nil
}
