package xs

import (
    "github.com/getkin/kin-openapi/openapi3"
    "github.com/jaswdr/faker"
)

func CreateValueMaker() ValueMaker {
    fake := faker.New()

    return func(namePath []string, schema *openapi3.Schema) any {
        for _, name := range namePath {
            if name == "id" {
                return fake.UUID()
            } else if name == "first" {
                return fake.Person().FirstName()
            } else if name == "last" {
                return fake.Person().LastName()
            } else if name == "age" {
                return 21
            }
        }
        return nil
    }
}
