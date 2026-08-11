.PHONY: direct aws gcp azure cloudflare migration migration-ui ui api mount clean

CMD = go run
MAIN = main.go
CLOUD_ROOT = cloud
WEB_ROOT = client/web
MIGRATION_ROOT = migration
PORT ?= 8080
FLATNAME ?= false
MODE ?= local
PORT_FLAG = -port=$(PORT)
FLATNAME_FLAG = -flat=$(FLATNAME)
MODE_FLAG = -mode=$(MODE)


direct:
	$(CMD) direct/$(MAIN)

aws:
	$(CMD) $(CLOUD_ROOT)/aws/$(MAIN)

cloudflare:
	$(CMD) $(CLOUD_ROOT)/cloudflare/$(MAIN)

gcp:
	$(CMD) $(CLOUD_ROOT)/gcp/$(MAIN)

azure:
	$(CMD) $(CLOUD_ROOT)/azure/$(MAIN)

# Headless local -> object-store migration demo (env-configured).
migration:
	$(CMD) $(MIGRATION_ROOT)/headless/$(MAIN)

# The full-featured UI (metrics, dedup, derivatives, events). The same binary
# also does migration — switch the backend with MODE=:
#   make ui                    # MODE=local (default) — filesystem
#   make ui MODE=migrate       # local -> object store dual-write + bulk copy
#   make ui MODE=r2            # object store only
# migrate/r2 need R2_ACCESS_KEY, R2_SECRET_KEY, R2_BUCKET, R2_ENDPOINT in the env.
ui:
	$(CMD) $(WEB_ROOT)/ui/$(MAIN) $(PORT_FLAG) $(FLATNAME_FLAG) $(MODE_FLAG)

# Convenience alias: the UI in migration mode (identical config, MODE=migrate).
migration-ui: MODE = migrate
migration-ui: ui

api:
	$(CMD) $(WEB_ROOT)/api/$(MAIN) $(PORT_FLAG)

mount:
	$(CMD) $(WEB_ROOT)/mount/$(MAIN) $(PORT_FLAG)

clean:
	rm -rf db.sqlite
	rm -rf media/