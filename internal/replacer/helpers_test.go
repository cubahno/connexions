package replacer

import (
	"github.com/jaswdr/faker/v2"
)

func newTestReplaceContext(schema any) *ReplaceContext {
	return &ReplaceContext{
		faker:  faker.New(),
		schema: schema,
	}
}
