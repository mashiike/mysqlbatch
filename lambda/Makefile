export AWS_ACCOUNT_ID := $(shell aws sts get-caller-identity --query 'Account' --output text)
.PHONY: clean deploy logs

clean:
	rm -f bootstrap
	rm -f *_dummy.zip

deploy: bootstrap
	lambroll --log-level debug deploy --tfstate .terraform/terraform.tfstate

.PHONY:
build:
	CGO_ENABLED=0 GOARCH=arm64 GOOS=linux go build -o bootstrap ../cmd/mysqlbatch/*.go

export MYSQLBATCH_VERSION=0.4.0
bootstrap:
	curl -sOL https://github.com/mashiike/mysqlbatch/releases/download/v$(MYSQLBATCH_VERSION)/mysqlbatch_$(MYSQLBATCH_VERSION)_linux_arm64.tar.gz && \
	mkdir tmp && \
	tar -zxvf mysqlbatch_$(MYSQLBATCH_VERSION)_linux_arm64.tar.gz -C tmp && \
	mv tmp/bootstrap ./bootstrap && \
	rm -rf tmp && \
	rm -rf mysqlbatch_$(MYSQLBATCH_VERSION)_linux_arm64.tar.gz

logs:
	lambroll logs --follow --format=short --tfstate .terraform/terraform.tfstate

.PHONY: terraform/init
terraform/init:
	terraform providers lock \
		-platform=darwin_amd64 \
		-platform=darwin_arm64 \
		-platform=linux_amd64 \
		-platform=linux_arm64 \
		-platform=windows_amd64
	terraform init


.PHONY: terraform/plan
terraform/plan:
	terraform plan


.PHONY: terraform/apply
terraform/apply:
	terraform apply

.PHONY: terraform/apply-refresh-only
terraform/apply-refresh-only:
	terraform apply -refresh-only
