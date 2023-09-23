### About
Services represent collection of resources available under the same first part of the path.<br/>
In case the path empty, the service name is empty too but in the `UI` it's seen as `/` and in the application code: `.root`.<br/>

Service can contain both OpenAPI resources and fixed resources.<br/>
Resources considered to be **OpenAPI** if their schema validates against either OpenAPI v3 or Swagger v2 schema.<br/>
**Fixed** is just a file that can be served as is.<br/><br/>

Resources without file extension should be stored with `index.json` filename.<br/>
See [File structure](#file-structure) for more details.<br/>

OpenAPI resources can be duplicated.<br/>
Fixed resources, on the contrary, can't be duplicated.<br/>

If the same resource is defined in both OpenAPI and fixed resources, the fixed resource will be used.<br/>
In the `UI` it's seen as `🔁` icon near the resource name.<br/>

Services are stored and served using file system.<br/>
There are no any database or any other storage.<br/>
Though in the future, local browser storage can be used to provide better experience.<br/>


### File structure

You can maintain this structure yourself or use `UI` to create and edit services and resources.<br/>
At any time you can import or export services and resources using `UI` or `API`.<br/>

#### OpenAPI resources
``` yaml
services
└───.openapi
    └───service-1.yml # (1)!
    └───service-2.json # (2)!
    └───service-3
        └───v1
            └───users
                └───index.yml # (3)!
            └───index.json # (4)!
```

1. `/*` all resources defined in service-1
2. `/*` all resources defined in service-2
3. `/service-3/v1/users/*` all resources defined in service-3
4. `/service-3/v1/*` all resources defined in service-3


#### Fixed resources without service name
``` yaml
services
└───.root
    └───get
        └───index.json # (1)!
    └───index.json # (2)!
    └───file.txt # (3)!
    └───patch
        └───service4
            └───v1
                └───users
                    └───index.json # (4)!
                └───me.json # (5)!
```

1. `GET /`
2. `GET /` (will overwrite the 1st one under `get`)
3. `GET /file.txt`
4. `PATCH /service-4/v1/users`
5. `PATCH /service-4/v1/me.json` (same as 4)


#### Fixed resources with service names
```yaml
services
└───service5
    └───delete
        └───users
            └───{user-id}
                └───index.json # (1)!
    └───get
        └───users
            └───index.json # (2)!
        └───file.txt # (3)!
    └───file-2.txt # (4)!
```

1. `DELETE /service-5/users/{user-id}`
2. `GET /service-5/users`
3. `GET /service-5/file.txt`
4. `GET /service-5/file-2.txt`
