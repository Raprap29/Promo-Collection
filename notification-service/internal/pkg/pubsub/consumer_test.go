package pubsub

import (
	"context"
	"errors"
	"notificationservice/internal/service/interfaces"
	"testing"

	"cloud.google.com/go/pubsub/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockMessage is a mock implementation of MessageInterface
type MockMessage struct {
	mock.Mock
	data []byte
}

func (m *MockMessage) Data() []byte {
	return m.data
}

func (m *MockMessage) Ack() {
	m.Called()
}

func (m *MockMessage) Nack() {
	m.Called()
}

// MockSubscriber is a mock implementation of SubscriberInterface
type MockSubscriber struct {
	mock.Mock
}

func (m *MockSubscriber) Receive(ctx context.Context, f func(context.Context, interfaces.MessageInterface)) error {
	args := m.Called(ctx, f)
	return args.Error(0)
}

// MockPubSubClient is a mock implementation of PubSubClientInterface
type MockPubSubClient struct {
	mock.Mock
}

func (m *MockPubSubClient) Subscriber(subscription string) interfaces.SubscriberInterface {
	args := m.Called(subscription)
	return args.Get(0).(interfaces.SubscriberInterface)
}

func (m *MockPubSubClient) Close() error {
	args := m.Called()
	return args.Error(0)
}

// MockPubSubClientFactory is a mock implementation of PubSubClientFactory
type MockPubSubClientFactory struct {
	mock.Mock
}

func (m *MockPubSubClientFactory) NewClient(ctx context.Context, projectID string) (interfaces.PubSubClientInterface, error) {
	args := m.Called(ctx, projectID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(interfaces.PubSubClientInterface), args.Error(1)
}

func TestNewPubSubConsumer(t *testing.T) {
	tests := []struct {
		name      string
		projectID string
		wantErr   bool
	}{
		{
			name:      "valid project ID",
			projectID: "test-project",
			wantErr:   false,
		},
		{
			name:      "empty project ID",
			projectID: "",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			consumer, err := NewPubSubConsumer(ctx, tt.projectID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, consumer)
			} else {
				assert.NoError(t, err)

				require.NotNil(t, consumer)
				require.NotNil(t, consumer.client)
				require.NotNil(t, consumer.ctx)
				require.NotNil(t, consumer.cancel)

				err = consumer.Close()
				assert.NoError(t, err)
			}
		})
	}
}

func TestNewPubSubConsumerWithFactory(t *testing.T) {
	tests := []struct {
		name      string
		projectID string
		setupMock func(*MockPubSubClientFactory)
		wantErr   bool
	}{
		{
			name:      "successful client creation",
			projectID: "test-project",
			setupMock: func(factory *MockPubSubClientFactory) {
				mockClient := &MockPubSubClient{}
				factory.On("NewClient", mock.Anything, "test-project").Return(mockClient, nil)
			},
			wantErr: false,
		},
		{
			name:      "factory error",
			projectID: "test-project",
			setupMock: func(factory *MockPubSubClientFactory) {
				factory.On("NewClient", mock.Anything, "test-project").Return(nil, errors.New("connection failed"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockFactory := &MockPubSubClientFactory{}
			tt.setupMock(mockFactory)

			consumer, err := NewPubSubConsumerWithFactory(ctx, tt.projectID, mockFactory)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, consumer)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, consumer)
				assert.NotNil(t, consumer.client)
				assert.NotNil(t, consumer.ctx)
				assert.NotNil(t, consumer.cancel)
			}

			mockFactory.AssertExpectations(t)
		})
	}
}

func TestPubSubConsumer_Consume(t *testing.T) {
	tests := []struct {
		name          string
		subscription  string
		setupMock     func(*MockPubSubClient, *MockSubscriber)
		handler       func(ctx context.Context, msg []byte) error
		expectedError bool
	}{
		{
			name:         "successful message processing",
			subscription: "test-subscription",
			setupMock: func(client *MockPubSubClient, subscriber *MockSubscriber) {
				subscriber.On("Receive", mock.Anything, mock.Anything).
					Run(func(args mock.Arguments) {
						// Simulate receiving a message
						handler := args.Get(1).(func(context.Context, interfaces.MessageInterface))
						msg := &MockMessage{data: []byte("test message")}
						msg.On("Ack").Return()
						handler(context.Background(), msg)
					}).
					Return(nil)
				client.On("Subscriber", "test-subscription").Return(subscriber)
			},
			handler: func(ctx context.Context, msg []byte) error {
				return nil // Success
			},
			expectedError: false,
		},
		{
			name:         "message processing error",
			subscription: "test-subscription",
			setupMock: func(client *MockPubSubClient, subscriber *MockSubscriber) {
				subscriber.On("Receive", mock.Anything, mock.Anything).
					Run(func(args mock.Arguments) {
						// Simulate receiving a message
						handler := args.Get(1).(func(context.Context, interfaces.MessageInterface))
						msg := &MockMessage{data: []byte("test message")}
						msg.On("Nack").Return()
						handler(context.Background(), msg)
					}).
					Return(nil)
				client.On("Subscriber", "test-subscription").Return(subscriber)
			},
			handler: func(ctx context.Context, msg []byte) error {
				return errors.New("processing failed")
			},
			expectedError: false,
		},
		{
			name:         "subscriber receive error",
			subscription: "test-subscription",
			setupMock: func(client *MockPubSubClient, subscriber *MockSubscriber) {
				subscriber.On("Receive", mock.Anything, mock.Anything).
					Return(errors.New("receive failed"))
				client.On("Subscriber", "test-subscription").Return(subscriber)
			},
			handler: func(ctx context.Context, msg []byte) error {
				return nil
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockClient := &MockPubSubClient{}
			mockSubscriber := &MockSubscriber{}

			tt.setupMock(mockClient, mockSubscriber)

			// Create consumer with proper structure
			consumerCtx, cancel := context.WithCancel(ctx)
			consumer := &PubSubConsumer{
				client: mockClient,
				ctx:    consumerCtx,
				cancel: cancel,
			}

			err := consumer.Consume(ctx, tt.subscription, tt.handler)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// Verify subscriber was stored
			assert.NotNil(t, consumer.sub)

			mockClient.AssertExpectations(t)
			mockSubscriber.AssertExpectations(t)
		})
	}
}

func TestPubSubConsumer_Unsubscribe(t *testing.T) {
	tests := []struct {
		name          string
		setupConsumer func() *PubSubConsumer
		expectedErr   error
	}{
		{
			name: "successful unsubscribe",
			setupConsumer: func() *PubSubConsumer {
				ctx, cancel := context.WithCancel(context.Background())
				return &PubSubConsumer{
					client: &MockPubSubClient{},
					ctx:    ctx,
					cancel: cancel,
				}
			},
			expectedErr: nil,
		},
		{
			name: "unsubscribe with nil cancel",
			setupConsumer: func() *PubSubConsumer {
				return &PubSubConsumer{
					client: &MockPubSubClient{},
					ctx:    context.Background(),
					cancel: nil,
				}
			},
			expectedErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			consumer := tt.setupConsumer()
			ctx := context.Background()

			err := consumer.Unsubscribe(ctx)

			if tt.expectedErr != nil {
				assert.Equal(t, tt.expectedErr.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}

			// Verify context was cancelled if cancel function exists
			if consumer.cancel != nil {
				select {
				case <-consumer.ctx.Done():
					// Context was cancelled, which is expected
				default:
					t.Error("Expected context to be cancelled after unsubscribe")
				}
			}
		})
	}
}

func TestPubSubConsumer_Close(t *testing.T) {
	tests := []struct {
		name        string
		setupMock   func(*MockPubSubClient)
		expectedErr error
	}{
		{
			name: "successful close",
			setupMock: func(client *MockPubSubClient) {
				client.On("Close").Return(nil)
			},
			expectedErr: nil,
		},
		{
			name: "close error",
			setupMock: func(client *MockPubSubClient) {
				client.On("Close").Return(errors.New("close failed"))
			},
			expectedErr: errors.New("close failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockPubSubClient{}
			tt.setupMock(mockClient)

			// Create consumer with proper structure
			ctx, cancel := context.WithCancel(context.Background())
			consumer := &PubSubConsumer{
				client: mockClient,
				ctx:    ctx,
				cancel: cancel,
			}

			err := consumer.Close()

			if tt.expectedErr != nil {
				assert.Equal(t, tt.expectedErr.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}

			// Verify context was cancelled
			select {
			case <-consumer.ctx.Done():
				// Context was cancelled, which is expected
			default:
				t.Error("Expected context to be cancelled after close")
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestDefaultMessage_WrapperLogic(t *testing.T) {
	realMsg := &pubsub.Message{Data: []byte("test message")}
	msgWrapper := &defaultMessage{msg: realMsg}

	data := msgWrapper.Data()
	assert.Equal(t, []byte("test message"), data)
	assert.NotPanics(t, func() {
		msgWrapper.Ack()
	})
	assert.NotPanics(t, func() {
		msgWrapper.Nack()
	})

	assert.NotNil(t, msgWrapper)
	assert.Equal(t, realMsg, msgWrapper.msg)
}

func TestWrapperCreation(t *testing.T) {
	realMsg := &pubsub.Message{Data: []byte("test")}
	msgWrapper := &defaultMessage{msg: realMsg}

	assert.Equal(t, realMsg.Data, msgWrapper.Data())
	assert.Implements(t, (*interfaces.MessageInterface)(nil), msgWrapper)
}

func TestMessageWrappingLogic(t *testing.T) {
	realMsg := &pubsub.Message{Data: []byte("test data")}
	msgWrapper := &defaultMessage{msg: realMsg}

	data := msgWrapper.Data()
	assert.Equal(t, []byte("test data"), data)

	assert.NotPanics(t, func() {
		msgWrapper.Ack()
	})

	assert.NotPanics(t, func() {
		msgWrapper.Nack()
	})

	assert.Implements(t, (*interfaces.MessageInterface)(nil), msgWrapper)
	assert.NotNil(t, msgWrapper.msg)
	assert.Equal(t, realMsg, msgWrapper.msg)
}

func TestDefaultPubSubClientFactory_NewClient_Error(t *testing.T) {
	factory := &defaultPubSubClientFactory{}
	ctx := context.Background()

	_, err := factory.NewClient(ctx, "")
	assert.Error(t, err)
}

func TestDefaultMessage_Ack_CallsRealMessageAck(t *testing.T) {
	realMsg := &pubsub.Message{Data: []byte("test")}
	wrapper := &defaultMessage{msg: realMsg}

	assert.NotPanics(t, func() {
		wrapper.Ack()
	})
}

func TestDefaultMessage_Nack_CallsRealMessageNack(t *testing.T) {
	realMsg := &pubsub.Message{Data: []byte("test")}
	wrapper := &defaultMessage{msg: realMsg}

	assert.NotPanics(t, func() {
		wrapper.Nack()
	})
}

func TestDefaultMessage_Data_AccessesRealMessageData(t *testing.T) {
	testData := []byte("test message data")
	realMsg := &pubsub.Message{Data: testData}
	wrapper := &defaultMessage{msg: realMsg}

	result := wrapper.Data()
	assert.Equal(t, testData, result)
}

func TestPubSubConsumer_Consume_SubscriberError(t *testing.T) {
	ctx := context.Background()
	mockClient := &MockPubSubClient{}

	// Test when Subscriber returns nil
	mockClient.On("Subscriber", "test-subscription").Return(nil)

	// Create consumer with proper structure
	consumerCtx, cancel := context.WithCancel(ctx)
	consumer := &PubSubConsumer{
		client: mockClient,
		ctx:    consumerCtx,
		cancel: cancel,
	}

	// This should panic when trying to call Receive on nil
	assert.Panics(t, func() {
		consumer.Consume(ctx, "test-subscription", func(ctx context.Context, msg []byte) error {
			return nil
		})
	})

	mockClient.AssertExpectations(t)
}
