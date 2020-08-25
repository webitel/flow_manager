
protoc -I ./workflow --go_out=plugins=grpc:./workflow ./workflow/*.proto

protoc -I ./client --go_out=plugins=grpc:./client ./client/*.proto