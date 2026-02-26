#!/bin/bash
set -x

PROTO_DIR="docreader/proto"
PYTHON_OUT="docreader/proto"
GO_OUT="docreader/proto"

python3 -m grpc_tools.protoc -I${PROTO_DIR} \
    --python_out=${PYTHON_OUT} \
    --pyi_out=${PYTHON_OUT} \
    --grpc_python_out=${PYTHON_OUT} \
    ${PROTO_DIR}/docreader.proto

protoc -I${PROTO_DIR} --go_out=${GO_OUT} \
    --go_opt=paths=source_relative \
    --go-grpc_out=${GO_OUT} \
    --go-grpc_opt=paths=source_relative \
    ${PROTO_DIR}/docreader.proto

if [ "$(uname)" == "Darwin" ]; then
    sed -i '' 's/import docreader_pb2/from docreader.proto import docreader_pb2/g' ${PYTHON_OUT}/docreader_pb2_grpc.py
else
    sed -i 's/import docreader_pb2/from docreader.proto import docreader_pb2/g' ${PYTHON_OUT}/docreader_pb2_grpc.py
fi

echo "Proto files generated successfully!"