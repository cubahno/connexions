package replacers

import (
	"github.com/jaswdr/faker"
)

func NewTestReplaceContext(schema any) *ReplaceContext {
	return &ReplaceContext{
		Faker:  faker.New(),
		Schema: schema,
	}
}
