package api

// @title           Infra Operator AWS API
// @version         1.0.1
// @description     API REST para gerenciamento de infraestrutura AWS
// @description     Suporta criação, atualização e deleção de recursos AWS via HTTP

// @contact.name   André Bassi
// @contact.url    https://github.com/andrebassi/infra-operator-aws

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:8080
// @BasePath  /api/v1

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name X-API-Key

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Bearer token (prefixo "Bearer " + API Key)
