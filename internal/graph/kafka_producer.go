package graph

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

// KafkaProducer publishes paper processing events to Kafka
type KafkaProducer struct {
	writer   *kafka.Writer
	topic    string
	enabled  bool
}

// PaperProcessedEvent represents a paper that has been processed
type PaperProcessedEvent struct {
	PaperTitle   string    `json:"paper_title"`
	LatexContent string    `json:"latex_content"`
	PDFPath      string    `json:"pdf_path"`
	ProcessedAt  time.Time `json:"processed_at"`
	Priority     int       `json:"priority"`
}

// NewKafkaProducer creates a new Kafka producer
func NewKafkaProducer(brokers []string, topic string, enabled bool) *KafkaProducer {
	if !enabled {
		log.Println("📊 Graph integration disabled (Kafka not enabled)")
		return &KafkaProducer{enabled: false}
	}

	writer := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		MaxAttempts:  3,
		BatchSize:    1,
		BatchTimeout: 10 * time.Millisecond,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		RequiredAcks: kafka.RequireOne,
		Async:        true, // Don't block paper processing
		Compression:  kafka.Snappy,
	}

	log.Printf("📡 Kafka producer initialized: %s -> topic: %s", brokers, topic)

	return &KafkaProducer{
		writer:  writer,
		topic:   topic,
		enabled: true,
	}
}

// PublishPaperProcessed publishes a paper.processed event to Kafka (non-blocking)
func (kp *KafkaProducer) PublishPaperProcessed(ctx context.Context, paperTitle, latexContent, pdfPath string) error {
	if !kp.enabled {
		// Kafka disabled, skip publishing
		return nil
	}

	event := PaperProcessedEvent{
		PaperTitle:   paperTitle,
		LatexContent: latexContent,
		PDFPath:      pdfPath,
		ProcessedAt:  time.Now(),
		Priority:     0,
	}

	// Marshal to JSON
	messageBytes, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Create Kafka message
	message := kafka.Message{
		Key:   []byte(paperTitle),
		Value: messageBytes,
		Time:  time.Now(),
	}

	// Publish asynchronously (non-blocking)
	go func() {
		writeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := kp.writer.WriteMessages(writeCtx, message)
		if err != nil {
			log.Printf("⚠️  Failed to publish to Kafka: %v (paper: %s)", err, paperTitle)
		} else {
			log.Printf("📤 Published to Kafka: %s", paperTitle)
		}
	}()

	return nil
}

// Close gracefully closes the Kafka producer
func (kp *KafkaProducer) Close() error {
	if !kp.enabled || kp.writer == nil {
		return nil
	}

	log.Println("🛑 Closing Kafka producer...")

	if err := kp.writer.Close(); err != nil {
		return fmt.Errorf("failed to close Kafka writer: %w", err)
	}

	log.Println("✅ Kafka producer closed")
	return nil
}

// GetStats returns producer statistics
func (kp *KafkaProducer) GetStats() kafka.WriterStats {
	if !kp.enabled || kp.writer == nil {
		return kafka.WriterStats{}
	}

	return kp.writer.Stats()
}
