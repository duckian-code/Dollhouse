.PHONY: build test validate-infra verify-s3-security verify-deployment verify-api-e2e verify-notification-redrive verify-monitoring-trace send-test-alert clean

COMMANDS := $(notdir $(wildcard cmd/*))

build:
	@mkdir -p .build
	@for command in $(COMMANDS); do \
		echo "building $$command"; \
		CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -buildvcs=false -o ".build/$$command/bootstrap" "./cmd/$$command" || exit 1; \
	done

test:
	go test ./...

validate-infra:
	sam validate --lint --template-file infrastructure/template.yaml

verify-s3-security:
	bash scripts/verify-s3-security.sh

verify-deployment:
	bash scripts/verify-deployment-config.sh

verify-api-e2e:
	bash scripts/verify-api-e2e.sh

verify-notification-redrive:
	@test -n "$(EVENT_ID)" || (echo "EVENT_ID is required" >&2; exit 2)
	bash scripts/verify-notification-redrive.sh "$(EVENT_ID)"

verify-monitoring-trace:
	@test -n "$(CORRELATION_ID)" || (echo "CORRELATION_ID is required" >&2; exit 2)
	bash scripts/verify-monitoring-trace.sh "$(CORRELATION_ID)"

send-test-alert:
	bash scripts/send-test-alert.sh

clean:
	rm -rf .build

# AWS SAM's makefile builder calls one target per function. Keeping the mapping
# here makes every template handler reproducible from the same Go module.
define BUILD_LAMBDA
build-$(2):
	@mkdir -p "$$(ARTIFACTS_DIR)"
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -buildvcs=false -o "$$(ARTIFACTS_DIR)/bootstrap" "./cmd/$(1)"
endef

$(eval $(call BUILD_LAMBDA,get-current-profile,GetCurrentProfileFunction))
$(eval $(call BUILD_LAMBDA,update-profile,UpdateProfileFunction))
$(eval $(call BUILD_LAMBDA,get-doll-configuration,GetDollConfigurationFunction))
$(eval $(call BUILD_LAMBDA,get-asset-catalog,GetAssetCatalogFunction))
$(eval $(call BUILD_LAMBDA,update-doll-configuration,UpdateDollConfigurationFunction))
$(eval $(call BUILD_LAMBDA,search-users,SearchUsersFunction))
$(eval $(call BUILD_LAMBDA,send-friend-request,SendFriendRequestFunction))
$(eval $(call BUILD_LAMBDA,list-friend-requests,ListFriendRequestsFunction))
$(eval $(call BUILD_LAMBDA,accept-friend-request,AcceptFriendRequestFunction))
$(eval $(call BUILD_LAMBDA,decline-friend-request,DeclineFriendRequestFunction))
$(eval $(call BUILD_LAMBDA,remove-friend,RemoveFriendFunction))
$(eval $(call BUILD_LAMBDA,publish-mood,PublishMoodFunction))
$(eval $(call BUILD_LAMBDA,get-friend-statuses,GetFriendStatusesFunction))
$(eval $(call BUILD_LAMBDA,notification-consumer,NotificationConsumerFunction))
