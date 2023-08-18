package xs

import "github.com/jaswdr/faker"

type FakeValue interface {
	~string | ~int | ~float64 | ~bool
}
type FakeFunc func() MixedValue

type MixedValue interface {
	Get() any
}

type StringValue string
type IntValue int
type BoolValue bool

func (s StringValue) Get() any {
	return string(s)
}

func (i IntValue) Get() any {
	return int(i)
}

func (b BoolValue) Get() any {
	return bool(b)
}

func AsString(f func() string) FakeFunc {
	return func() MixedValue {
		return StringValue(f())
	}
}

func GetFakes() map[string]FakeFunc {
	fake := faker.New()
	person := fake.Person()
	pet := fake.Pet()

	return map[string]FakeFunc{
		"person.name":              AsString(person.Name),
		"person.first_name":        AsString(person.FirstName),
		"person.last_name":         AsString(person.LastName),
		"person.gender":            AsString(person.Gender),
		"person.first_name_female": AsString(person.FirstNameFemale),
		"person.first_name_male":   AsString(person.FirstNameMale),
		"person.gender_female":     AsString(person.GenderFemale),
		"person.gender_male":       AsString(person.GenderMale),
		"person.name_female":       AsString(person.NameFemale),
		"person.name_male":         AsString(person.NameMale),
		"person.ssn":               AsString(person.SSN),
		"person.suffix":            AsString(person.Suffix),
		"person.title":             AsString(person.Title),
		"person.title_female":      AsString(person.TitleFemale),
		"person.title_male":        AsString(person.TitleMale),

		"pet.name": AsString(pet.Name),
		"pet.dog":  AsString(pet.Dog),
		"pet.cat":  AsString(pet.Cat),
	}
}
