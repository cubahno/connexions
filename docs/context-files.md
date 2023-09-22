## Contexts

Contexts are organized in YAML files which act more like namespaces or collections of contexts.<br/>
Usually, file names should correspond to the name of the service or the domain.<br/>
For example, `payments`, or `petstore`.<br/><br/>

On the filesystem, contexts stored with the provided name and `yml` extension in the `contexts` directory.<br/>
For example, `contexts/payments.yml`.<br/><br/>

Only individual primitive properties replaced during content generation.<br/>
So, you cannot substitute property with an object or a list.<br/>

### Contexts structure

Inside the context file you should provide the data that corresponds to your schema.<br/>
For example, if you have an OpenAPI schema with the following structure:

```yaml
Pet:
  type: object
  properties:
    id:
      type: string
      format: uuid
    name:
      type: string
    tag:
      type: string
```

Our `petstore.yml` context file could look like this:
```yaml
id: 123e4567-e89b-12d3-a456-426614174000
name: "doggie"
tag: "dog"
```

So, in case of `json` response, the replacement will look like this:
```json
{
  "id": "123e4567-e89b-12d3-a456-426614174000",
  "name": "doggie",
  "tag": "dog"
}
```

Any Schema that has these properties will be replaced with the provided values.<br/>
So, the provided context is more like hardcoded-values.<br/>

Let's look at another Schema with nested properties:

```yaml
PetWithOwner:
  type: object
  properties:
  pet:
    type: object
    properties:
      id:
        type: string
        format: uuid
      name:
        type: string
      tag:
        type: string
    ownerPerson:
      type: object
      properties:
        id:
          type: integer
        name:
          type: string
```

It's clear that using name as `doggie` is not ideal here.<br/>
Our context file could look like this:

```yaml
pet:
  id: 123e4567-e89b-12d3-a456-426614174000
  name: "doggie"
  tag: "dog"
owner_person:
  id: 1
  name: "Jane Doe"
``` 
    
The keys inside context files should be `snake_case`.<br/>

Our response would look like:<br/>
```json
{
    "pet": {
        "id": "123e4567-e89b-12d3-a456-426614174000",
        "name": "doggie",
        "tag": "dog"
    },
    "owner_person": {
        "id": 1,
        "name": "Jane Doe"
    }
}
```

### Dynamic keys and values
We use the `fake` library to generate random values.<br/>
You can use it inside your context files to generate dynamic values:
```yaml
pet:
  id: "fake:uuid4"
  name: "fake:pet.name"
  tag: "fake:gamer.tag"
owner_person:
  id: "fake:u_int8"
  name: "fake:person.name"
``` 

All available `fakes` are shipped with the distribution.<br/>
 `contexts/fakes` file has the [full list](https://github.com/cubahno/connexions/blob/master/resources/contexts/fake.yml).<br/>

**The syntax** is `fake:path.with.dot`.<br/>

It is possible to use not only dynamic values but dynamic keys as well:
```yaml
_id$: "alias:fake.u_int8"
_email$: "alias:fake.internet.email"
```

Any property in the schema that ends with `_id`, `_email`, (in case of camelCase `Id`, `Email`) in its name will be replaced with the generated value.<br/>
For example:
```json
{
  "userId": 123,
  "userEmail": "jane.doe@example.com"
}
```

### Context reuse
It is possible to reuse contexts in other contexts.<br/>
Let's say we have context files with the following content:<br/><br/>

```yaml title="petstore.yml"
id: "fake:uuid4"
name: "fake:pet.name"
tag: "fake:gamer.tag"
```

```yaml title="person.yml"
id: "fake:u_int8"
name: "fake:person.name"
```

We can refer to any property in any context file using `alias` keyword:

```yaml title="petstore.yml"
id: "fake:uuid4"
name: "fake:pet.name"
tag: "fake:gamer.tag"
owner_id: "alias:person.id"
owner_name: "alias:person.name"
```

```yaml title="person.yml"
id: "fake:u_int8"
name: "fake:person.name"
pet_id: "alias:petstore.id"
pet_name: "alias:petstore.name"
```

**The syntax** is `alias:<context-file>.<dotted.path>`.<br/>
Some commonly used alias located in `contexts/common` [file](https://github.com/cubahno/connexions/blob/master/resources/contexts/common.yml) 
<br/>

If the `alias` won't point to any target property, it will be used as-is, so you can notice the issue.<br/>


### Other keywords
Along with `alias` and `fake` keywords, there are some other keywords that can be used in context files.<br/>

#### `botify`
Replaces value with the letters `???` and the number `###`.<br/>
For example,
```yaml
password: "botify:???###"
```
will produce a string like `abc123`.<br/>

#### `func`
Allows to use custom functions to generate values.<br/>
There are currently no such functions but they can appear here when needed.<br/>


### Predefined values
To replace property only with predefined set of values we can use list of values instead of single value.<br/>

```yaml title="person.yml"
name: ["Jane", "John"]
```

or 
```yaml
name:
    - "Jane"
    - "John"
```
both notations are valid YAML.<br/>
Random value will be picked from a list in this case.<br/>


### Path and Headers
Values replacement for `path` and `headers` can be used with different dedicated context.<br/>

```yaml
in-path:
  pet_id: "alias:fake.u_int8"
  id$: "alias:fake.u_int8"
  
in-header:
  x_pet_name: "alias:fake.pet.name"
```

These contexts will takes precedence over the default context replacements.<br/>
The `in-` prefix can be changed in the `config.yml` file:
```yaml
app:
  # ...
  contextAreaPrefix: "in-"
```


### Wiring
The filenames are completely arbitrary, and there's no magic involved in regards to which contexts are used for any particular service.<br/>
You need to set the corresponding configuration manually in the config file or using the UI.<br/><br/>
Each distribution ships with the defaults contexts, in case there's no configuration for service provided - default will be used.<br/><br/>

Let's say we have a `petstore` service and 2 contexts:

```yaml title="fake.yml"
pet:
  dog: "fake:"
  cat: "fake:"
  name: "fake:"
payments:
  # ... mappings
people:
  # ... mappings
```

```yaml title="person.yml"
id: "fake:u_int8"
name: "fake:person.name"
```

Fake file is unnecessary big, we can use just a portion of it:<br/>
```yaml title="config.yml"
services:
  petstore:
   # the name of the context to use when substituting the values in the request/response.
    contexts:
      - fake: pet
      - fake: people
      - person:
```


Only 2 maps will be taken from `fake` context and complete `person` context will be used.<br/>
Replacement will be applied in the order of definition.<br/>


### Using in Fixed Responses
Contexts can be used in fixed responses as well.<br/>
Fixed response is the file with contents that you provided, usually with a `json` contents, for example:
```json
{
  "id": 123,
  "name": "doggie",
  "tag": "dog"
}
```

In order to use contexts in fixed responses, we would need to wrap the values in `{}` brackets regardless of the value type.<br/>
```json
{
  "id": "{id}",
  "name": "{name}",
  "tag": "{tag}"
}
```

With a context of:
```yaml
id: 123
name: "doggie"
```

it will be replaced with:
```json
{
  "id": 123,
  "name": "doggie",
  "tag": "some-string-value"
}
```

The types will be correctly resolved as well: `id` is unsigned int8.<br/>

It's possible to use multiple placeholders in a single value:
```json
{
  "id": 123,
  "nameWithId": "{name}-{id}"
}
```

will be replaced with:
```json
{
  "id": 123,
  "nameWithId": "doggie-123"
}
```
