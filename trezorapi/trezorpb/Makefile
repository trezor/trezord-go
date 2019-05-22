combine:
	cd trezor-common/protob && make combine

messages:
	protoc --proto_path=trezor-common/protob/ --go_out=messages,import_path=trezorpb:. trezor-common/protob/combined.proto
	mv combined.pb.go messages.pb.go

marshalb:
	go run ./meta/main.go > marshal/marshal.go
