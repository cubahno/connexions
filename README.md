<div style="text-align: center; width:450px;">
    <img src="./resources/docs/images/gotham.svg">
</div>

## Connexions

[![CI](https://github.com/cubahno/connexions/workflows/CI/badge.svg?event=push)](https://github.com/cubahno/connexions/actions/workflows/ci.yml?query=event%3Apush+branch%3Amaster+workflow%3ACI)
[![Cov](https://img.shields.io/endpoint?url=https://gist.githubusercontent.com/cubahno/4110782af3ec09dd1ebabc3304756f1f/raw/covbadge.json)](https://github.com/cubahno/connexions/actions/workflows/ci.yml?query=event%3Apush+branch%3Amaster+workflow%3ACI)


**Connexions** is a library originally inspired by [Connexion](https://github.com/spec-first/connexion).<br/>
Connexion allows you to set up a REST API with Swagger documentation and OAuth2 authentication with minimal effort.<br/>

Connexions takes this one step further by allowing you to define **multiple APIs** not limited to only Swagger and(or) OpenAPI.<br/>
You can define single response for any arbitrary path on the fly.<br/>

## Goal:<br/>
 - simplify the development process

## Features:<br/>
- Randomized response contents, allowing you to redefine the response for any path in a locale of your choice
- Mimic error responses and status codes
- Configurable latency in responses

## Simple start:<br/>

```bash 

docker run -it --rm \
  -p 2200:2200 \
  -v connexions:/app/resources \
  cubahno/connexions api

``` 

License
===================
Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License. You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0.

Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.


Links
===================
[OpenAPI Specification](https://www.openapis.org/)<br/>
[OpenAPI 3.0 Style Values](https://github.com/OAI/OpenAPI-Specification/blob/master/versions/3.0.2.md#style-values)<br/>
[Operation Object](https://github.com/swagger-api/swagger-spec/blob/master/versions/2.0.md#operation-object)<br/>
[YAML format](https://github.com/OAI/OpenAPI-Specification/blob/master/versions/2.0.md#format)<br/>
