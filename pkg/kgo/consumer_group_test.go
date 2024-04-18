package kgo

import (
	"context"
	"strconv"
	"testing"
)

func TestIssue686(t *testing.T) {
	consumeTopic, cleanup := tmpTopic(t)
	defer cleanup()
	produceTopic, cleanup := tmpTopic(t)
	defer cleanup()

	errs := make(chan error)
	body := []byte(randsha()) // a small body so we do not flood RAM

	eagerConsumer := newTestConsumer(
		errs,
		consumeTopic,
		produceTopic,
		"topic-group",
		[]GroupBalancer{RangeBalancer()},
		body,
	)
	cooperativeEagerConsumer := newTestConsumer(
		errs,
		consumeTopic,
		produceTopic,
		"topic-group",
		[]GroupBalancer{CooperativeStickyBalancer(), RangeBalancer()},
		body,
	)

	p, err := newTestClient(
		DefaultProduceTopic(consumeTopic),
	)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 10000; i++ {
		p.Produce(context.TODO(), &Record{Value: body, Key: []byte(strconv.Itoa(1))}, nil)
	}
	if err := p.Flush(context.TODO()); err != nil {
		t.Fatal(err)
	}

	cooperativeEagerConsumer.goRun(false, 10)
	eagerConsumer.goRun(false, 3)

	doneConsume := make(chan struct{})
	go func() {
		cooperativeEagerConsumer.wait()
		eagerConsumer.wait()
		close(doneConsume)
	}()

	for {
		select {
		case <-doneConsume:
			return
		case err := <-errs:
			t.Fatal(err)
		}
	}
}
