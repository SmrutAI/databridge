module github.com/SmrutAI/databridge

go 1.25.0

require (
	github.com/Azure/azure-sdk-for-go/sdk/storage/azblob v1.4.0
	github.com/SmrutAI/smritea-sdk/go v0.0.0
	github.com/aws/aws-lambda-go v1.47.0
	github.com/aws/aws-sdk-go-v2 v1.32.0
	github.com/aws/aws-sdk-go-v2/config v1.27.0
	github.com/aws/aws-sdk-go-v2/service/s3 v1.61.0
	github.com/awslabs/aws-lambda-go-api-proxy v0.16.0
	github.com/knights-analytics/hugot v0.6.5
	github.com/labstack/echo/v4 v4.12.0
	github.com/pgvector/pgvector-go v0.2.2
	github.com/qdrant/go-client v1.17.0
	github.com/smacker/go-tree-sitter v0.0.0-20240827094217-dd81d9e9be82
	github.com/tushar2708/conveyor v0.0.0
	gorm.io/driver/postgres v1.5.7
	gorm.io/gorm v1.25.7
)

require (
	github.com/Azure/azure-sdk-for-go/sdk/azcore v1.13.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/internal v1.10.0 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.6.4 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.17.0 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.15.0 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.3.16 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.6.16 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.0 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.3.16 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.11.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.3.18 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.11.18 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.17.16 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.19.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.22.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.27.0 // indirect
	github.com/aws/smithy-go v1.22.0 // indirect
	github.com/daulet/tokenizers v1.26.0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/emirpasic/gods v1.18.1 // indirect
	github.com/go-errors/errors v1.5.1 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/gofrs/flock v0.13.0 // indirect
	github.com/golang-jwt/jwt v3.2.2+incompatible // indirect
	github.com/gomlx/exceptions v0.0.3 // indirect
	github.com/gomlx/go-huggingface v0.3.2 // indirect
	github.com/gomlx/go-xla v0.2.0 // indirect
	github.com/gomlx/gomlx v0.27.1 // indirect
	github.com/gomlx/onnx-gomlx v0.4.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgx/v5 v5.7.0 // indirect
	github.com/jackc/puddle/v2 v2.2.1 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/knights-analytics/ortgenai v0.1.0 // indirect
	github.com/labstack/gommon v0.4.2 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mitchellh/colorstring v0.0.0-20190213212951-d06e56a500db // indirect
	github.com/patrickmn/go-cache v2.1.0+incompatible // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/satori/go.uuid v1.2.0 // indirect
	github.com/schollz/progressbar/v2 v2.15.0 // indirect
	github.com/sugarme/regexpset v0.0.0-20200920021344-4d4ec8eaf93c // indirect
	github.com/sugarme/tokenizer v0.3.0 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasttemplate v1.2.2 // indirect
	github.com/viant/afs v1.30.0 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	github.com/yalue/onnxruntime_go v1.27.0 // indirect
	golang.org/x/crypto v0.49.0 // indirect
	golang.org/x/exp v0.0.0-20260312153236-7ab1446f8b90 // indirect
	golang.org/x/image v0.37.0 // indirect
	golang.org/x/net v0.51.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	golang.org/x/term v0.41.0 // indirect
	golang.org/x/text v0.35.0 // indirect
	golang.org/x/time v0.14.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260209200024-4cfbd4190f57 // indirect
	google.golang.org/grpc v1.78.0 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
	k8s.io/klog/v2 v2.140.0 // indirect
)

// Local development replace directives — both conveyor and smritea-sdk/go are
// git submodules in the parent smritea-cloud repo. These must be in go.mod (not
// only go.work) so that `go mod tidy` uses the local path directly without
// attempting VCS resolution for the placeholder v0.0.0 versions.
replace (
	github.com/SmrutAI/smritea-sdk/go => ./../../smritea-sdk/go
	github.com/tushar2708/conveyor => ./../conveyor
)
