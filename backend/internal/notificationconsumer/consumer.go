package notificationconsumer

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/dollhouse-app/dollhouse/backend/internal/observability"
)

const (
	currentSchemaVersion = 1
	maxExpoBatchSize     = 100
)

// Consumer processes SQS notification jobs and reports record-level failures.
type Consumer struct {
	store  Store
	push   PushService
	logger *slog.Logger
}

// New creates a notification consumer.
func New(store Store, push PushService, logger *slog.Logger) *Consumer {
	if logger == nil {
		logger = slog.Default()
	}
	return &Consumer{store: store, push: push, logger: logger}
}

// Handle acknowledges the successful FIFO prefix and returns the failed and
// unprocessed suffix so retries preserve queue order.
func (c *Consumer) Handle(ctx context.Context, event events.SQSEvent) (events.SQSEventResponse, error) {
	failures := make([]events.SQSBatchItemFailure, 0)
	for index, record := range event.Records {
		if err := c.processRecord(ctx, record); err != nil {
			c.logger.ErrorContext(ctx, "notification job failed",
				"lifecycle", "failed", "messageId", record.MessageId, "error", err)
			for _, unprocessed := range event.Records[index:] {
				failures = append(failures, events.SQSBatchItemFailure{ItemIdentifier: unprocessed.MessageId})
			}
			break
		}
	}
	return events.SQSEventResponse{BatchItemFailures: failures}, nil
}

func (c *Consumer) processRecord(ctx context.Context, record events.SQSMessage) error {
	job, err := decodeJob(record.Body)
	if err != nil {
		return err
	}
	ctx = observability.WithRequest(ctx, observability.RequestContext{
		CorrelationID: job.CorrelationID,
		RequestID:     record.MessageId,
	}, c.logger)
	logger := observability.Logger(ctx)
	logger.InfoContext(ctx, "notification job started",
		"lifecycle", "started", "messageId", record.MessageId,
		"eventId", job.EventID,
		"recipientCount", len(job.RecipientUserIDs))

	devices, err := c.store.ListEnabledDevices(ctx, job.RecipientUserIDs)
	if err != nil {
		return fmt.Errorf("list enabled devices: %w", err)
	}
	if len(devices) == 0 {
		logger.InfoContext(ctx, "notification job completed",
			"lifecycle", "completed", "messageId", record.MessageId,
			"eventId", job.EventID, "attempted", 0, "accepted", 0, "invalidated", 0)
		emitNotificationMetrics(ctx, 0, 0, 0)
		return nil
	}

	accepted, invalidated := 0, 0
	for start := 0; start < len(devices); start += maxExpoBatchSize {
		end := min(start+maxExpoBatchSize, len(devices))
		batch := devices[start:end]
		messages := make([]Message, len(batch))
		for i, device := range batch {
			messages[i] = Message{
				To: device.PushToken, Title: "Dollhouse", Body: "A friend shared a new mood.", Sound: "default",
				Data: map[string]any{"type": "mood_updated", "eventId": job.EventID},
			}
		}
		tickets, err := c.push.Send(ctx, messages)
		if err != nil {
			return fmt.Errorf("send Expo batch: %w", err)
		}
		if len(tickets) != len(batch) {
			return fmt.Errorf("Expo returned %d tickets for %d messages", len(tickets), len(batch))
		}
		for i, ticket := range tickets {
			switch {
			case ticket.Status == "ok" && ticket.ID != "":
				accepted++
			case ticket.Status == "error" && ticket.Details.Error == "DeviceNotRegistered":
				if err := c.store.DisableDevice(ctx, batch[i]); err != nil {
					return fmt.Errorf("disable invalid device: %w", err)
				}
				invalidated++
			default:
				return fmt.Errorf("Expo rejected notification: status=%q code=%q", ticket.Status, ticket.Details.Error)
			}
		}
	}

	logger.InfoContext(ctx, "notification job completed",
		"lifecycle", "completed", "messageId", record.MessageId,
		"eventId", job.EventID, "attempted", len(devices),
		"accepted", accepted, "invalidated", invalidated)
	emitNotificationMetrics(ctx, len(devices), accepted, invalidated)
	return nil
}

func emitNotificationMetrics(ctx context.Context, attempted, succeeded, invalidated int) {
	observability.Emit(ctx,
		observability.Metric{Name: "NotificationJobsProcessed", Value: 1, Unit: "Count"},
		observability.Metric{Name: "NotificationDeliveryAttempts", Value: float64(attempted), Unit: "Count"},
		observability.Metric{Name: "NotificationDeliverySuccesses", Value: float64(succeeded), Unit: "Count"},
		observability.Metric{Name: "InvalidDeviceTokens", Value: float64(invalidated), Unit: "Count"},
	)
}

func decodeJob(body string) (Job, error) {
	decoder := json.NewDecoder(bytes.NewBufferString(body))
	decoder.DisallowUnknownFields()
	var job Job
	if err := decoder.Decode(&job); err != nil {
		return Job{}, fmt.Errorf("decode notification job: %w", err)
	}
	if err := ensureJSONEnd(decoder); err != nil {
		return Job{}, err
	}
	if job.SchemaVersion != currentSchemaVersion {
		return Job{}, fmt.Errorf("unsupported notification schema version %d", job.SchemaVersion)
	}
	if strings.TrimSpace(job.EventID) == "" || strings.TrimSpace(job.SenderUserID) == "" ||
		strings.TrimSpace(job.CorrelationID) == "" || strings.TrimSpace(job.CreatedAt) == "" {
		return Job{}, errors.New("notification job identifiers are incomplete")
	}
	seen := make(map[string]struct{}, len(job.RecipientUserIDs))
	recipients := make([]string, 0, len(job.RecipientUserIDs))
	for _, recipient := range job.RecipientUserIDs {
		recipient = strings.TrimSpace(recipient)
		if recipient == "" {
			return Job{}, errors.New("notification job contains an empty recipient ID")
		}
		if _, exists := seen[recipient]; !exists {
			seen[recipient] = struct{}{}
			recipients = append(recipients, recipient)
		}
	}
	job.RecipientUserIDs = recipients
	return job, nil
}

func ensureJSONEnd(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("notification job must contain one JSON object")
		}
		return fmt.Errorf("decode notification job: %w", err)
	}
	return nil
}
