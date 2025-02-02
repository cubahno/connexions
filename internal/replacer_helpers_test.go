package internal

import (
	"github.com/jaswdr/faker/v2"
)

func NewTestReplaceContext(schema any) *ReplaceContext {
	return &ReplaceContext{
		Faker:  faker.New(),
		Schema: schema,
	}
}
