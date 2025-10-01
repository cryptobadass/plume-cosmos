#!/usr/bin/env bash

set -eo pipefail

protoc_gen_gocosmos() {
  if ! grep "github.com/gogo/protobuf => github.com/regen-network/protobuf" go.mod &>/dev/null ; then
    echo -e "\tPlease run this command from somewhere inside the cosmos-sdk folder."
    return 1
  fi

  go get github.com/regen-network/cosmos-proto/protoc-gen-gocosmos@latest 2>/dev/null
}

protoc_gen_gocosmos
PATH="${PATH}:${HOME}/go/bin"

proto_files=$(find ./proto -type f -name '*.proto' | sort)
for file in $proto_files; do
  buf generate \
    --template proto/buf.gen.yaml \
    --path "${file}"
done

# command to generate docs using protoc-gen-doc
# buf protoc \
#   -I "proto" \
#   -I "third_party/proto" \
#   --doc_out=./docs/core \
#   --doc_opt=./docs/protodoc-markdown.tmpl,proto-docs.md \
#   $(find "$(pwd)/proto" -maxdepth 5 -name '*.proto')
go mod tidy

# generate codec/testdata proto code
# buf generate \
#   --template proto/buf.gen.yaml \
#   --path "testutil/testdata"

# move proto files to the right places
cp -r github.com/cosmos/cosmos-sdk/* ./
rm -rf github.com
