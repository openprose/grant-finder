.PHONY: validate validate-product-cli dogfood-agent

validate:
	cd cli/grant-finder && go test ./...

validate-product-cli:
	cd cli/grant-finder && go build -o /tmp/grant-finder ./cmd/grant-finder
	python3 scripts/validate_product_surface.py --check-cli /tmp/grant-finder

dogfood-agent: validate-product-cli
	python3 scripts/validate_agent_dogfood.py --binary /tmp/grant-finder
