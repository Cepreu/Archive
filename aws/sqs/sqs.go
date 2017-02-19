package sqs

import (
	"strconv"

	"github.com/WorkFit/commongo/polling"
	"github.com/WorkFit/go/errors"
	"github.com/WorkFit/go/log"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
)

// Message represents a message in a message queue.
type Message struct {
	// Body is the message's contents (not URL-encoded).
	Body string
	// Handle is an identifier associated with the act of receiving the message.
	// A new receipt handle is returned every time you receive a message.
	// When deleting a message, provide the last received receipt handle.
	Handle string
}

// MessageQueue represents a message queue.
type MessageQueue interface {
	polling.Receiver
	MessageDeleter
}

// MessageDeleter deletes a batch of messages from the queue.
type MessageDeleter interface {
	// DeleteMessages deletes a batch of messages from the queue.
	DeleteMessages(handles []string) error
}

type queue struct {
	*sqs.SQS
	*sqs.ReceiveMessageInput
	receiveMessage func(*sqs.ReceiveMessageInput) (*sqs.ReceiveMessageOutput, error) `test-hook:"verify-unexported"`
}

const (
	nonExistentQueueErrorCode = "AWS.SimpleQueueService.NonExistentQueue"
)

var (
	awsConfig = aws.NewConfig().WithRegion("us-west-2")
)

// NewMessageQueue creates a new SQS message queue.
func NewMessageQueue(queueURL string) MessageQueue {
	r := &queue{
		SQS: sqs.New(session.New(awsConfig)),
		ReceiveMessageInput: &sqs.ReceiveMessageInput{
			QueueUrl:            aws.String(queueURL),
			MaxNumberOfMessages: aws.Int64(10),
			WaitTimeSeconds:     aws.Int64(20),
		},
	}
	r.receiveMessage = r.ReceiveMessage
	return r
}

// Receive receives a batch of messages from the queue.
// By grouping messages into batches, we can reduce our Amazon SQS costs.
func (q *queue) Receive() (interface{}, bool, error) {
	response, err := q.receiveMessage(q.ReceiveMessageInput)

	if err != nil {
		awsError := err.(awserr.Error)

		if awsError.Code() == nonExistentQueueErrorCode {
			log.Error(nonExistentQueueErrorCode, "awsError", awsError, "input", q.ReceiveMessageInput)
			panic(awsError)
		}
	}

	return adaptMessages(response.Messages), len(response.Messages) > 0, err
}

// DeleteMessages deletes a batch of messages from the queue.
// By grouping messages into batches, we can reduce our Amazon SQS costs.
func (q *queue) DeleteMessages(handles []string) error {
	entries := make([]*sqs.DeleteMessageBatchRequestEntry, len(handles))
	for i, handle := range handles {
		entries[i] = &sqs.DeleteMessageBatchRequestEntry{
			Id:            aws.String(strconv.Itoa(i)),
			ReceiptHandle: aws.String(handle),
		}
	}

	input := &sqs.DeleteMessageBatchInput{QueueUrl: q.ReceiveMessageInput.QueueUrl, Entries: entries}
	output, err := q.DeleteMessageBatch(input)

	if err == nil && len(output.Failed) > 0 {
		return errors.WF11201(handles, output)
	}

	return err
}

// adaptMessages converys SQS messages (a vendored data type) into objects
// of type Message.
func adaptMessages(input []*sqs.Message) []*Message {
	output := make([]*Message, len(input))
	for i, message := range input {
		output[i] = &Message{Body: *message.Body, Handle: *message.ReceiptHandle}
	}
	return output
}
