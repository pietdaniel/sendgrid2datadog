package main

import (
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"testing"
)

func TestEventUnmarshal(t *testing.T) {
	assert := assert.New(t)

	dat, err := ioutil.ReadFile("./webhook_fixtures/bounce_event.json")
	assert.NoError(err)
	var events SendGridEvents
	err = eventUnmarshal(dat, &events)
	assert.NoError(err)
	assert.Len(events, 1)
	event := events[0]

	assert.Equal(event.Email, "example@test.com")
	// Categories can be arrays or strings depending on your SendGrid configuration
	// see: https://sendgrid.com/docs/for-developers/tracking-events/event/#json-objects
	assert.Equal(event.Category, []string{"cat facts"})
	assert.Equal(event.Reason, "500 unknown recipient")

	dat0, err := ioutil.ReadFile("./webhook_fixtures/deferred_event.json")
	var events0 SendGridEvents
	assert.NoError(err)
	err = eventUnmarshal(dat0, &events0)
	assert.NoError(err)
	assert.Len(events0, 1)
	event = events0[0]

	assert.Equal("example@test.com", event.Email)
	// Categories can be arrays or strings depending on your SendGrid configuration
	// see: https://sendgrid.com/docs/for-developers/tracking-events/event/#json-objects
	assert.Equal([]string{"cat facts"}, event.Category)
	assert.Equal("", event.Reason)
	assert.Equal("400 try again later", event.Response)
}
