FILE=coverage.out
COVER_EXCLUDE=".pb.go|/pkg/grpcx/|app.go|.gen.go"

go test --count=1 --covermode=count --coverprofile=$FILE --coverpkg=./... ./...
if [[ "$COVER_EXCLUDE" != "" ]]; then grep -Ev "$COVER_EXCLUDE" $FILE > $FILE.tmp && mv $FILE.tmp $FILE ; fi

go tool cover --func=$FILE
go tool cover -html=$FILE