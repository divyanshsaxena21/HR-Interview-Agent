package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

type EvaluationJob struct {
	InterviewID string `json:"interview_id"`
	EnqueuedAt  int64  `json:"enqueued_at"`
	Attempt     int    `json:"attempt"`
}

type EvalQueue struct {
	client       *redis.Client
	queueKey     string
	useMemory    bool
	memoryQueue  chan *EvaluationJob
	memoryMutex  sync.Mutex
}

const (
	evalQueueKey = "eval_jobs"
	maxRetries   = 3
	jobTimeout   = 30 * time.Second
)

func NewEvalQueue() *EvalQueue {
	redisURL := os.Getenv("REDIS_URL")
	
	// If REDIS_URL is empty or not set, use in-memory queue
	if redisURL == "" {
		log.Println("Redis not configured; using in-memory evaluation queue")
		return &EvalQueue{
			useMemory:   true,
			memoryQueue: make(chan *EvaluationJob, 100),
		}
	}

	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Printf("Warning: invalid REDIS_URL %s, falling back to in-memory queue: %v", redisURL, err)
		return &EvalQueue{
			useMemory:   true,
			memoryQueue: make(chan *EvaluationJob, 100),
		}
	}

	client := redis.NewClient(opts)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = client.Ping(ctx).Result()
	if err != nil {
		log.Printf("Warning: Redis connection failed: %v (falling back to in-memory queue)", err)
		return &EvalQueue{
			useMemory:   true,
			memoryQueue: make(chan *EvaluationJob, 100),
		}
	}

	log.Println("Redis connected; using Redis evaluation queue")
	return &EvalQueue{
		client:   client,
		queueKey: evalQueueKey,
		useMemory: false,
	}
}

func (eq *EvalQueue) EnqueueJob(interviewID string) error {
	if eq.useMemory {
		job := &EvaluationJob{
			InterviewID: interviewID,
			EnqueuedAt:  time.Now().Unix(),
			Attempt:     0,
		}
		select {
		case eq.memoryQueue <- job:
			return nil
		default:
			return fmt.Errorf("memory queue full")
		}
	}

	if eq.client == nil {
		return fmt.Errorf("redis client not available")
	}

	job := EvaluationJob{
		InterviewID: interviewID,
		EnqueuedAt:  time.Now().Unix(),
		Attempt:     0,
	}

	jobJSON, _ := json.Marshal(job)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return eq.client.RPush(ctx, eq.queueKey, jobJSON).Err()
}

func (eq *EvalQueue) DequeueJob(ctx context.Context) (*EvaluationJob, error) {
	if eq.useMemory {
		select {
		case job := <-eq.memoryQueue:
			return job, nil
		case <-time.After(5 * time.Second):
			return nil, nil // timeout
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	if eq.client == nil {
		return nil, fmt.Errorf("redis client not available")
	}

	// BLPOP with 5 second timeout
	result, err := eq.client.BLPop(ctx, 5*time.Second, eq.queueKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // timeout, try again
		}
		return nil, err
	}

	if len(result) < 2 {
		return nil, fmt.Errorf("unexpected redis response")
	}

	jobJSON := result[1]
	var job EvaluationJob
	if err := json.Unmarshal([]byte(jobJSON), &job); err != nil {
		return nil, err
	}

	return &job, nil
}

func (eq *EvalQueue) RequeueJob(job *EvaluationJob) error {
	job.Attempt++
	if job.Attempt > maxRetries {
		log.Printf("Job %s exceeded max retries, discarding", job.InterviewID)
		return nil
	}

	if eq.useMemory {
		select {
		case eq.memoryQueue <- job:
			return nil
		default:
			return fmt.Errorf("memory queue full")
		}
	}

	if eq.client == nil {
		return fmt.Errorf("redis client not available")
	}

	jobJSON, _ := json.Marshal(job)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Re-enqueue with a small delay (push back to queue)
	return eq.client.RPush(ctx, eq.queueKey, jobJSON).Err()
}

func (eq *EvalQueue) Close() error {
	if eq.client != nil {
		return eq.client.Close()
	}
	return nil
}

func (eq *EvalQueue) IsAvailable() bool {
	if eq.useMemory {
		return true // in-memory queue is always available
	}
	if eq.client == nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err := eq.client.Ping(ctx).Result()
	return err == nil
}
