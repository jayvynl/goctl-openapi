package constant

const (
	// traditional go-zero info keys
	ApiInfoTitle   = "title"
	ApiInfoDesc    = "desc"
	ApiInfoVersion = "version"
	ApiInfoAuthor  = "author"
	ApiInfoEmail   = "email"
	// extended openapi keys
	// https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.0.3.md#fixed-fields
	ApiInfoServers      = "servers"      // comma separated urls
	ApiInfoExternalDocs = "externalDocs" // url
	ApiInfoTags         = "tags"         // comma separated string

	OptionDefault   = "default"
	OptionOptional  = "optional"
	OptionOptions   = "options"
	OptionRange     = "range"
	OptionOmitempty = "omitempty"

	TagKeyHeader = "header"
	TagKeyPath   = "path"
	TagKeyForm   = "form"
	TagKeyJson   = "json"
	// https://github.com/go-playground/validator
	TagKeyValidate = "validate"
)
