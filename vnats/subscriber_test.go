package vnats

import (
	"fmt"
	"reflect"
	"testing"
	"time"
)

type subscribeStringsConfig struct {
	publishMessages  []string
	expectedMessages []string
	mode             SubscriptionMode
	wantErr          bool
}

var subscriberTestCases = []subscribeStringsConfig{
	{[]string{"hello", "world"}, []string{"hello", "world"}, SingleSubscriberStrictMessageOrder, false},
	{[]string{"hello", "world"}, []string{"world", "hello"}, SingleSubscriberStrictMessageOrder, true},
	{[]string{"hello", "world"}, []string{"hello", "world"}, MultipleSubscribersAllowed, false},
	{[]string{"hello", "world"}, []string{"world", "hello"}, MultipleSubscribersAllowed, false},
	{[]string{"hello", "world"}, []string{"world"}, MultipleSubscribersAllowed, true},
	{[]string{"hello", "world"}, []string{"world"}, SingleSubscriberStrictMessageOrder, true},
}

func TestSubscriber_Subscribe_Strings(t *testing.T) {
	for _, test := range subscriberTestCases {
		subject := integrationTestStreamName + ".PubSubTest"

		conn := makeIntegrationTestConn(t, integrationTestStreamName, log)

		for idx, msg := range test.publishMessages {
			pub, err := conn.NewPublisher(NewPublisherArgs{
				StreamName: integrationTestStreamName,
				Encoding:   EncJSON,
			})
			if err != nil {
				t.Error(err)
			}

			if err := pub.Publish(PublishArgs{
				Subject: subject,
				MsgID:   fmt.Sprintf("msg-%d", idx),
				Data:    msg,
			}); err != nil {
				t.Error(err)
			}
		}
		receivedMessages, err := retrieveStringMessages(conn, subject, test.mode)
		if err != nil {
			t.Error(err)
		}
		switch test.mode {
		case MultipleSubscribersAllowed:
			if err := cmpStringSlicesIgnoreOrder(test.expectedMessages, receivedMessages); err != nil && !test.wantErr {
				t.Error(err)
			} else if test.wantErr && err == nil {
				t.Error("Should fail, but no error was thrown!")
			}
		case SingleSubscriberStrictMessageOrder:
			equal := reflect.DeepEqual(receivedMessages, test.expectedMessages)
			if !equal && !test.wantErr {
				t.Errorf("Got %v, expected %v\n", receivedMessages, test.expectedMessages)
			} else if equal && test.wantErr {
				t.Error("Should fail, but no error was thrown!")
			}
		}

	}
}

func cmpStringSlicesIgnoreOrder(expectedMessages []string, receivedMessages []string) error {
	for _, expectedMsg := range expectedMessages {
		for idx, foundMsg := range receivedMessages {
			if expectedMsg == foundMsg {
				receivedMessages[idx] = receivedMessages[len(receivedMessages)-1]
				receivedMessages = receivedMessages[:len(receivedMessages)-1]
			}
		}

	}
	if !reflect.DeepEqual(receivedMessages, []string{}) {
		return fmt.Errorf("more messages were received than published. Additional msgs: %v", receivedMessages)
	}
	return nil
}

func retrieveStringMessages(conn Connection, subject string, mode SubscriptionMode) ([]string, error) {
	sub, err := conn.NewSubscriber(NewSubscriberArgs{
		ConsumerName: "TestConsumer",
		Subject:      subject,
		Encoding:     EncJSON,
		Mode:         mode,
	})
	if err != nil {
		return nil, err
	}
	var receivedMessages []string
	handler := func(msg string) error {
		receivedMessages = append(receivedMessages, msg)
		return nil
	}

	if err := sub.Subscribe(handler); err != nil {
		return nil, err
	}
	time.Sleep(time.Second) // sleep to retrieve all messages
	return receivedMessages, nil
}
