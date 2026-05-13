FUZZTIME ?= 10s

.PHONY: validate validate-product-cli dogfood-agent fuzz-smoke secret-scan

validate:
	cd cli/grant-finder && go test ./...

validate-product-cli:
	cd cli/grant-finder && go build -o /tmp/grant-finder ./cmd/grant-finder
	python3 scripts/validate_product_surface.py --check-cli /tmp/grant-finder

dogfood-agent: validate-product-cli
	python3 scripts/validate_agent_dogfood.py --binary /tmp/grant-finder

fuzz-smoke:
	cd cli/grant-finder && go test -run=^$$ -fuzz=FuzzParseAssignment -fuzztime=$(FUZZTIME) ./internal/grantfinder
	cd cli/grant-finder && go test -run=^$$ -fuzz=FuzzParseFeedItems -fuzztime=$(FUZZTIME) ./internal/grantfinder
	cd cli/grant-finder && go test -run=^$$ -fuzz=FuzzParseGrantsXMLZip -fuzztime=$(FUZZTIME) ./internal/grantfinder
	cd cli/grant-finder && go test -run=^$$ -fuzz=FuzzOpportunityFromFederalRegister -fuzztime=$(FUZZTIME) ./internal/grantfinder
	cd cli/grant-finder && go test -run=^$$ -fuzz=FuzzSelectJSONFields -fuzztime=$(FUZZTIME) ./internal/cli
	cd cli/grant-finder && go test -run=^$$ -fuzz=FuzzEnsureReadOnlySQL -fuzztime=$(FUZZTIME) ./internal/cli

secret-scan:
	gitleaks detect --source . --redact --no-banner
	gitleaks detect --source . --no-git --redact --no-banner
