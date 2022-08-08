
.PHONY: help run build install license example
all: help

## license: Adds license header to missing files.
license:
	@echo "  >  \033[32mAdding license headers...\033[0m "
	GO111MODULE=off go get -u github.com/google/addlicense
	addlicense -c "ChainSafe Systems" -f ./scripts/header.txt -y 2021 .

## license-check: Checks for missing license headers
license-check:
	@echo "  >  \033[Checking for license headers...\033[0m "
	GO111MODULE=off go get -u github.com/google/addlicense
	addlicense -check -c "ChainSafe Systems" -f ./scripts/header.txt -y 2021 .


coverage:
	go tool cover -func cover.out | grep total | awk '{print $3}'

test:
	./scripts/tests.sh

## Install dependency subkey
install-subkey:
	curl https://getsubstrate.io -sSf | bash -s -- --fast
	cargo install --force --git https://github.com/paritytech/substrate subkey

genmocks:
	mockgen -destination=./tss/common/mock/tss.go github.com/binance-chain/tss-lib/tss Message
	mockgen -destination=./tss/common/mock/communication.go -source=./tss/common/base.go -package mock_tss
	mockgen -destination=./tss/keygen/mock/storer.go -source=./tss/keygen/keygen.go
	mockgen -destination=./tss/keygen/mock/storer.go -source=./tss/keygen/keygen.go
	mockgen --package mock_tss -destination=./tss/mock/storer.go -source=./tss/resharing/resharing.go
	mockgen -source=./tss/coordinator.go -destination=./tss/mock/coordinator.go
	mockgen -source=./comm/communication.go -destination=./comm/mock/communication.go
	mockgen -source=./chains/evm/listener/event-handler.go -destination=./chains/evm/listener/mock/listener.go
	mockgen -destination=chains/evm/listener/mock/deposit-handler.go github.com/ChainSafe/sygma-core/chains/evm/listener DepositHandler
	mockgen -source=./chains/evm/calls/events/listener.go -destination=./chains/evm/calls/events/mock/listener.go

e2e-test:
	./scripts/e2e_tests.sh

example:
	docker-compose --file=./example/docker-compose.yml up
