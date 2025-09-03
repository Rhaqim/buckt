.PHONY: direct aws gcp azure ui api mount

CMD = go run
MAIN = main.go
CLOUD_ROOT = cloud
WEB_ROOT = client/web
PORT ?= 8080
PORT_FLAG = -port=$(PORT)

direct:
	$(CMD) direct/$(MAIN)

aws:
	$(CMD) $(CLOUD_ROOT)/aws/$(MAIN)

gcp:
	$(CMD) $(CLOUD_ROOT)/gcp/$(MAIN)

azure:
	$(CMD) $(CLOUD_ROOT)/azure/$(MAIN)

ui:
	$(CMD) $(WEB_ROOT)/UI/$(MAIN) $(PORT_FLAG)

api:
	$(CMD) $(WEB_ROOT)/api/$(MAIN) $(PORT_FLAG)

mount:
	$(CMD) $(WEB_ROOT)/mount/$(MAIN) $(PORT_FLAG)