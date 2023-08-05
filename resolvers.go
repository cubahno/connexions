package xs

import (
    "github.com/brianvoe/gofakeit/v6"
    "github.com/getkin/kin-openapi/openapi3"
)

type Resolver struct {
}

func CreateValueMaker() ValueMaker {
    faker := gofakeit.New(0)

    return func(schema *openapi3.Schema, state *GeneratorState) any {
        namePath := state.NamePath
        for _, name := range namePath {
            if name == "id" {
                return faker.UUID()
            } else if name == "first" {
                return faker.Person().FirstName
            } else if name == "last" {
                return faker.Person().LastName
            } else if name == "age" {
                return 21
            } else if name == "name" {
                return faker.PetName()
            } else if name == "tag" {
                return faker.Gamertag()
            }
        }

        return state.Example
    }
}
