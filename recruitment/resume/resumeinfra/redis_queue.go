package resumeinfra

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Abraxas-365/relay/pkg/kernel"
	"github.com/Abraxas-365/relay/recruitment/resume"
	"github.com/redis/go-redis/v9"
)

// RedisQueue implements JobQueue interface using Redis
type RedisQueue struct {
	client    *redis.Client
	queueName string
}

// NewRedisQueue creates a new Redis-based queue
func NewRedisQueue(client *redis.Client, queueName string) resume.JobQueue {
	return &RedisQueue{
		client:    client,
		queueName: queueName,
	}
}

// Enqueue adds a job to the queue
func (q *RedisQueue) Enqueue(ctx context.Context, jobID kernel.JobID, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload for job %s: %w", jobID, err)
	}

	if err := q.client.LPush(ctx, q.queueName, data).Err(); err != nil {
		return fmt.Errorf("enqueue job %s: %w", jobID, err)
	}

	return nil
}

// Dequeue gets a job from the queue (blocking with timeout)
func (q *RedisQueue) Dequeue(ctx context.Context, timeout time.Duration) ([]byte, error) {
	result, err := q.client.BRPop(ctx, timeout, q.queueName).Result()
	if err != nil {
		// redis.Nil is returned when timeout occurs
		if err == redis.Nil {
			return nil, nil // No jobs available
		}
		return nil, fmt.Errorf("dequeue job: %w", err)
	}

	if len(result) < 2 {
		return nil, fmt.Errorf("invalid result from queue: expected 2 elements, got %d", len(result))
	}

	return []byte(result[1]), nil
}

// EnqueueDelayed schedules a job for later processing (for retries)
func (q *RedisQueue) EnqueueDelayed(ctx context.Context, jobID kernel.JobID, payload any, delay time.Duration) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal delayed payload for job %s: %w", jobID, err)
	}

	score := float64(time.Now().Add(delay).Unix())
	delayedQueue := q.queueName + ":delayed"

	if err := q.client.ZAdd(ctx, delayedQueue, redis.Z{
		Score:  score,
		Member: data,
	}).Err(); err != nil {
		return fmt.Errorf("enqueue delayed job %s: %w", jobID, err)
	}

	return nil
}

// MoveDelayedToReady moves delayed jobs that are ready to the main queue
func (q *RedisQueue) MoveDelayedToReady(ctx context.Context) (int, error) {
	delayedQueue := q.queueName + ":delayed"
	now := float64(time.Now().Unix())

	// Get jobs ready to process
	jobs, err := q.client.ZRangeByScore(ctx, delayedQueue, &redis.ZRangeBy{
		Min: "-inf",
		Max: fmt.Sprintf("%f", now),
	}).Result()

	if err != nil {
		return 0, fmt.Errorf("get delayed jobs: %w", err)
	}

	if len(jobs) == 0 {
		return 0, nil
	}

	// Use pipeline for atomic operations
	pipe := q.client.Pipeline()
	for _, job := range jobs {
		pipe.LPush(ctx, q.queueName, job)
		pipe.ZRem(ctx, delayedQueue, job)
	}

	if _, err := pipe.Exec(ctx); err != nil {
		return 0, fmt.Errorf("move delayed jobs to ready: %w", err)
	}

	return len(jobs), nil
}

// GetQueueSize returns the number of jobs in the queue
func (q *RedisQueue) GetQueueSize(ctx context.Context) (int64, error) {
	size, err := q.client.LLen(ctx, q.queueName).Result()
	if err != nil {
		return 0, fmt.Errorf("get queue size: %w", err)
	}
	return size, nil
}

// GetDelayedQueueSize returns the number of delayed jobs
func (q *RedisQueue) GetDelayedQueueSize(ctx context.Context) (int64, error) {
	delayedQueue := q.queueName + ":delayed"
	size, err := q.client.ZCard(ctx, delayedQueue).Result()
	if err != nil {
		return 0, fmt.Errorf("get delayed queue size: %w", err)
	}
	return size, nil
}

// Clear removes all jobs from the queue (use with caution - for testing/maintenance)
func (q *RedisQueue) Clear(ctx context.Context) error {
	pipe := q.client.Pipeline()
	pipe.Del(ctx, q.queueName)
	pipe.Del(ctx, q.queueName+":delayed")

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("clear queue: %w", err)
	}

	return nil
}

// Ping checks if Redis connection is alive
func (q *RedisQueue) Ping(ctx context.Context) error {
	return q.client.Ping(ctx).Err()
}

// GetStats returns queue statistics
func (q *RedisQueue) GetStats(ctx context.Context) (map[string]any, error) {
	ready, err := q.GetQueueSize(ctx)
	if err != nil {
		return nil, err
	}

	delayed, err := q.GetDelayedQueueSize(ctx)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"queue_name":   q.queueName,
		"ready_jobs":   ready,
		"delayed_jobs": delayed,
		"total_jobs":   ready + delayed,
	}, nil
}

