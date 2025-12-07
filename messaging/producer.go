package messaging

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

// Writer adalah koneksi kita ke Kafka
var writer *kafka.Writer

// InitKafkaProducer membuka koneksi ke Kafka
func InitKafkaProducer(brokerUrl string, topic string) {
	writer = &kafka.Writer{
		Addr:     kafka.TCP(brokerUrl),
		Topic:    topic,
		Balancer: &kafka.LeastBytes{}, // Menyeimbangkan beban
	}
	log.Println("✅ Kafka Producer siap di topic:", topic)
}

// PublishEvent mengirim data apa saja ke Kafka
func PublishEvent(action string, kecamatanID uint, data interface{}) {
	// Struktur pesan yang akan dikirim
	message := map[string]interface{}{
		"action":       action, // Contoh: "CREATE_BENCANA"
		"kecamatan_id": kecamatanID,
		"timestamp":    time.Now(),
		"payload":      data, // Data detailnya
	}

	// Ubah ke JSON
	jsonPayload, _ := json.Marshal(message)

	// Kirim ke Kafka
	err := writer.WriteMessages(context.Background(),
		kafka.Message{
			Key:   []byte(action), // Key opsional
			Value: jsonPayload,
		},
	)

	if err != nil {
		log.Printf("❌ Gagal kirim event ke Kafka: %v", err)
	} else {
		log.Printf("📤 Event Terkirim: %s", action)
	}
}
