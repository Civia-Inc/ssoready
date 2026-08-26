module github.com/Civia-Inc/ssoready

go 1.25.0

require (
	connectrpc.com/connect v1.20.0
	connectrpc.com/vanguard v0.4.0
	github.com/alecthomas/participle/v2 v2.1.4
	github.com/aws/aws-sdk-go-v2 v1.43.8
	github.com/aws/aws-sdk-go-v2/config v1.32.38
	github.com/aws/aws-sdk-go-v2/service/s3 v1.107.4
	github.com/aws/aws-sdk-go-v2/service/secretsmanager v1.44.7
	github.com/aws/smithy-go v1.27.10
	github.com/cloudflare/cloudflare-go v0.117.0
	github.com/cyrusaf/ctxlog v1.3.3
	github.com/getsentry/sentry-go v0.48.0
	github.com/go-jose/go-jose/v3 v3.0.5
	github.com/go-jose/go-jose/v4 v4.1.4
	github.com/golang-jwt/jwt/v5 v5.3.1
	github.com/golang-migrate/migrate/v4 v4.19.1
	github.com/google/go-cmp v0.7.0
	github.com/google/uuid v1.6.0
	github.com/gorilla/mux v1.8.1
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.30.0
	github.com/hasura/go-graphql-client v0.16.0
	github.com/jackc/pgx/v5 v5.10.0
	github.com/resend/resend-go/v2 v2.28.0
	github.com/rs/cors v1.11.1
	github.com/segmentio/analytics-go/v3 v3.3.0
	github.com/ssoready/conf v0.0.0-20240508183332-dbc356674c9e
	github.com/ssoready/prettyuuid v0.0.0-20241023163822-285da46017b3
	github.com/stretchr/testify v1.12.1
	github.com/ucarion/cli v0.2.0
	golang.org/x/crypto v0.54.0
	google.golang.org/genproto/googleapis/api v0.0.0-20260803160001-6ac0973c030d
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260803160001-6ac0973c030d
	google.golang.org/grpc v1.83.0
	google.golang.org/protobuf v1.36.12
)

require (
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.7.19 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.19.37 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.18.38 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.39 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.39 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.4.40 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.18 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.9.32 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.13.39 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.19.40 // indirect
	github.com/aws/aws-sdk-go-v2/service/signin v1.5.7 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.33.7 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.38.7 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.45.7 // indirect
	github.com/bmizerany/assert v0.0.0-20160611221934-b7ed37b82869 // indirect
	github.com/coder/websocket v1.8.14 // indirect
	github.com/goccy/go-json v0.10.5 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/lib/pq v1.10.9 // indirect
	github.com/segmentio/backo-go v1.0.0 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	golang.org/x/net v0.57.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.40.0 // indirect
	golang.org/x/time v0.12.0 // indirect
)

replace github.com/ssoready/conf => github.com/Civia-Inc/conf v0.0.0-20240508183332-dbc356674c9e

replace github.com/ssoready/prettyuuid => github.com/Civia-Inc/prettyuuid v0.0.0-20241023163822-285da46017b3
