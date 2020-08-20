
protoc -I ./flow --go_out=plugins=grpc:./flow ./flow/*.proto

protoc -I ./client --go_out=plugins=grpc:./client ./client/*.proto