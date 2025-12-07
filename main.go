package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)

var ctx = context.Background()

type Destination struct {
	Host  string `json:"host"`
	Port  string `json:"port"`
	Topic string `json:"topic"`
}

type Task struct {
	ID              string      `json:"id"`
	Attempt         int         `json:"attempt"`
	Source          string      `json:"source"`
	Destination     Destination `json:"destination"`
	DeadDestination Destination `json:"dead_destination"`
	MaxRetries      int         `json:"max_retries"`
	BaseDelay       int         `json:"base_delay"`
	ClientId        string      `json:"client_id"`
	IsPriority      bool        `json:"is_priority"`
	MessageData     string      `json:"message_data"`
	DestinationType string      `json:"destination_type"`
}

const (
	retryKey = "retry:schedule:"
	// baseDelay    = 2 * time.Second
	// maxRetries   = 1
	pollInterval = 1 * time.Second
)

func handleAddTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var task Task
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Initialize attempt count if not set
	task.Attempt = 0

	// Schedule the task immediately
	if err := scheduleTask(redisClient, task, 0); err != nil {
		http.Error(w, fmt.Sprintf("Failed to schedule task: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"message": fmt.Sprintf("Task %s scheduled successfully", task.ID),
	})
}

var redisClient *redis.Client

func main() {
	redisClient = NewRedisClient()
	if err := PingRedis(redisClient); err != nil {
		fmt.Println("Failed to connect to Redis:", err)
		return
	}
	fmt.Println("Connected to Redis successfully!")

	// Optional: seed initial tasks
	// seedTasks(redisClient)

	// Start the worker in a goroutine
	go worker(redisClient)

	// Setup HTTP server
	http.HandleFunc("/tasks", handleAddTask)

	// Start HTTP server
	fmt.Println("Starting HTTP server on :8080...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func scheduleTask(rdb *redis.Client, task Task, delay time.Duration) error {
	data, err := json.Marshal(task)
	if err != nil {
		return err
	}

	score := float64(time.Now().Add(delay).Unix())
	return rdb.ZAdd(ctx, retryKey, redis.Z{
		Score:  score,
		Member: data,
	}).Err()
}

func worker(rdb *redis.Client) {
	for {
		now := float64(time.Now().Unix())
		// Fetch up to N tasks whose time has come
		tasks, err := rdb.ZRangeByScoreWithScores(ctx, retryKey, &redis.ZRangeBy{
			Min:    "0",
			Max:    fmt.Sprintf("%f", now),
			Offset: 0,
			Count:  10,
		}).Result()
		if err != nil {
			log.Printf("Error fetching tasks: %v\n", err)
			time.Sleep(pollInterval)
			continue
		}

		if len(tasks) == 0 {
			time.Sleep(pollInterval)
			continue
		}

		for _, z := range tasks {
			var task Task
			success := true
			err := json.Unmarshal([]byte(z.Member.(string)), &task)
			if err != nil {
				log.Printf("Error unmarshaling task: %v\n", err)
				continue
			}
			fmt.Printf("Processing task %s (attempt %d)\n", task.ID, task.Attempt)
			// Remove task from queue
			rdb.ZRem(ctx, retryKey, z.Member)

			fmt.Printf("Processing task %s (attempt %d)\n", task.ID, task.Attempt)
			//========================
			// Here you would process the task, e.g., send a message to Kafka
			if task.DestinationType == "kafka" {
				// Send message to Kafka
				kafkaClient, err := NewKafkaClient(task.Destination.Host + ":" + task.Destination.Port)
				if err != nil {
					log.Printf("Error creating Kafka client: %v\n", err)
					continue
				}
				defer kafkaClient.Close()

				// Produce message
				time.Sleep(5000 * time.Millisecond)
				if err := kafkaClient.PushMessage(ctx, task.Destination.Topic, []byte(fmt.Sprintf("%s|%d", task.ID, task.Attempt)), []byte(task.MessageData)); err != nil {
					log.Printf("Error producing message to Kafka: %v\n", err)
					success = false
				}
			}
			//========================
			// Simulate task failure
			// success := false // for demo
			if !success {
				task.Attempt++
				fmt.Println("Attempt: ", task.Attempt)
				fmt.Println("Max Retries: ", task.MaxRetries)
				if task.Attempt > task.MaxRetries {
					log.Printf("Task %s failed on %dth attempts. Giving up.\n", task.ID, task.Attempt)
					continue
				}

				// Schedule retry with exponential backoff
				delay := time.Duration(float64(task.BaseDelay*int(time.Second)) * math.Pow(2, float64(task.Attempt-1)))
				fmt.Println("Scheduling retry for task:", task.ID, "with delay:", delay)
				err := scheduleTask(rdb, task, delay)
				if err != nil {
					log.Printf("Failed to reschedule task %s: %v\n", task.ID, err)
				} else {
					log.Printf("Rescheduled task %s with delay %v\n", task.ID, delay)
				}
			} else {
				log.Printf("Task %s completed successfully\n", task.ID)
			}
		}
	}
}

func seedTasks(rdb *redis.Client) {
	dest := Destination{Host: "localhost", Port: "9092", Topic: "kafka-topic"}
	destDead := Destination{Host: "localhost", Port: "9092", Topic: "dead-kafka-topic"}
	tasks := []Task{
		{ID: "task-1", Attempt: 0, Source: "test-app", Destination: dest, DeadDestination: destDead, MaxRetries: 3, BaseDelay: 1, ClientId: "123", IsPriority: true, MessageData: "Email body structure", DestinationType: "kafka"},
		{ID: "task-2", Attempt: 1, Source: "test-app2", Destination: dest, DeadDestination: destDead, MaxRetries: 2, BaseDelay: 2, ClientId: "234", IsPriority: false, MessageData: "Email body structure2", DestinationType: "kafka"},
		{ID: "task-3", Attempt: 2, Source: "test-app3", Destination: dest, DeadDestination: destDead, MaxRetries: 1, BaseDelay: 3, ClientId: "345", IsPriority: true, MessageData: "Email body structure3", DestinationType: "kafka"},
	}

	for _, task := range tasks {
		_ = scheduleTask(rdb, task, 0)
	}
}

/*
Processing task task-1 (attempt 0)
2025/08/19 23:17:13 Rescheduled task task-1 with delay 1s
Processing task task-2 (attempt 1)
2025/08/19 23:17:13 Rescheduled task task-2 with delay 4s
Processing task task-3 (attempt 2)
2025/08/19 23:17:13 Task task-3 failed after 3 attempts. Giving up.
Processing task task-1 (attempt 1)
2025/08/19 23:17:14 Rescheduled task task-1 with delay 2s
Processing task task-1 (attempt 2)
2025/08/19 23:17:16 Rescheduled task task-1 with delay 4s
Processing task task-2 (attempt 2)
2025/08/19 23:17:17 Rescheduled task task-2 with delay 8s
Processing task task-1 (attempt 3)
2025/08/19 23:17:20 Task task-1 failed after 4 attempts. Giving up.
Processing task task-2 (attempt 3)
2025/08/19 23:17:25 Rescheduled task task-2 with delay 16s
Processing task task-2 (attempt 4)
2025/08/19 23:17:41 Rescheduled task task-2 with delay 32s
Processing task task-2 (attempt 5)
2025/08/19 23:18:13 Task task-2 failed after 6 attempts. Giving up.
*/

/*
localhost:6379> zrange retry:schedule 0 -1 withscores
1) "{\"id\":\"task-1\",\"attempt\":1,\"source\":\"test-app\",\"destination\":\"kafka-topic\",\"dead_destination\":\"dead-kafka-topic\",\"max_retries\":3,\"base_delay\":1,\"client_id\":\"123\",\"is_priority\":true,\"message_data\":\"Email body structure\"}"
2) "1755625907"
3) "{\"id\":\"task-2\",\"attempt\":2,\"source\":\"test-app2\",\"destination\":\"kafka-topic\",\"dead_destination\":\"dead-kafka-topic\",\"max_retries\":5,\"base_delay\":2,\"client_id\":\"234\",\"is_priority\":false,\"message_data\":\"Email body structure2\"}"
4) "1755625910"



localhost:6379> zrange retry:schedule 0 -1 withscores
1) "{\"id\":\"task-1\",\"attempt\":2,\"source\":\"test-app\",\"destination\":\"kafka-topic\",\"dead_destination\":\"dead-kafka-topic\",\"max_retries\":3,\"base_delay\":1,\"client_id\":\"123\",\"is_priority\":true,\"message_data\":\"Email body structure\"}"
2) "1755625909"
3) "{\"id\":\"task-2\",\"attempt\":2,\"source\":\"test-app2\",\"destination\":\"kafka-topic\",\"dead_destination\":\"dead-kafka-topic\",\"max_retries\":5,\"base_delay\":2,\"client_id\":\"234\",\"is_priority\":false,\"message_data\":\"Email body structure2\"}"
4) "1755625910"



localhost:6379> zrange retry:schedule 0 -1 withscores
1) "{\"id\":\"task-2\",\"attempt\":2,\"source\":\"test-app2\",\"destination\":\"kafka-topic\",\"dead_destination\":\"dead-kafka-topic\",\"max_retries\":5,\"base_delay\":2,\"client_id\":\"234\",\"is_priority\":false,\"message_data\":\"Email body structure2\"}"
2) "1755625910"
3) "{\"id\":\"task-1\",\"attempt\":3,\"source\":\"test-app\",\"destination\":\"kafka-topic\",\"dead_destination\":\"dead-kafka-topic\",\"max_retries\":3,\"base_delay\":1,\"client_id\":\"123\",\"is_priority\":true,\"message_data\":\"Email body structure\"}"
4) "1755625913"



localhost:6379> zrange retry:schedule 0 -1 withscores
1) "{\"id\":\"task-2\",\"attempt\":3,\"source\":\"test-app2\",\"destination\":\"kafka-topic\",\"dead_destination\":\"dead-kafka-topic\",\"max_retries\":5,\"base_delay\":2,\"client_id\":\"234\",\"is_priority\":false,\"message_data\":\"Email body structure2\"}"
2) "1755625918"


localhost:6379> zrange retry:schedule 0 -1 withscores
1) "{\"id\":\"task-2\",\"attempt\":3,\"source\":\"test-app2\",\"destination\":\"kafka-topic\",\"dead_destination\":\"dead-kafka-topic\",\"max_retries\":5,\"base_delay\":2,\"client_id\":\"234\",\"is_priority\":false,\"message_data\":\"Email body structure2\"}"
2) "1755625918"

*/
// zadd retry:schedule 1755623982 "{\"ID\":\"task-7\",\"attempt\":1,\"source\":\"test-app7\",\"destination\":\"kafka-topic\",\"dead_destination\":\"dead-kafka-topic\",\"max_retries\":5,\"base_delay\":2,\"client_id\":\"234\",\"is_priority\":false,\"message_data\":\"Email body structure5\"}"
