.PHONY: proto

proto: ## Generate protobuf Go + Python stubs
	cd proto && buf lint
	cd proto && buf generate
