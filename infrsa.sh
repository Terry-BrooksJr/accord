#!/usr/bin/env bash
set -euo pipefail

# Run from the empty directory that will become the Accord repo.
# The scaffold script itself may be present.
script_name="${0##*/}"

shopt -s dotglob nullglob
for entry in *; do
  if [[ "$entry" != "$script_name" ]]; then
    printf 'Directory must be empty except for %s.\n' "$script_name" >&2
    exit 1
  fi
done
shopt -u dotglob nullglob

directories=(
  services/webhook-gateway/cmd/gateway
  services/webhook-gateway/internal/{config,http,signatures,idempotency,journal,storage,observability}
  services/webhook-gateway/tests

  services/integration-api/src/accord_api/{routes,auth,schemas,dependencies}
  services/integration-api/tests

  services/integration-worker/src/accord_worker/{jobs,workflows,retries,outbox,reconciliation}
  services/integration-worker/tests

  packages/python/accord-core/src/accord_core/{config,models,storage,events,observability}
  packages/python/accord-core/tests

  packages/python/accord-integrations/src/accord_integrations/{base,stripe,shippo,plaid,zendesk,mock_support}
  packages/python/accord-integrations/tests

  database/postgresql/{migrations,seeds}
  database/clickhouse/{migrations,queries}

  etl/src/accord_etl/{extract,transform,load,checkpoints,quality}
  etl/tests

  infrastructure/docker
  infrastructure/traefik/dynamic
  infrastructure/postgresql
  infrastructure/clickhouse
  infrastructure/keycloak/realms
  infrastructure/prometheus/{rules,targets}
  infrastructure/grafana/provisioning/{dashboards,datasources}
  infrastructure/grafana/dashboards
  infrastructure/loki

  docs/{adr,api,integrations,runbooks}
  scenarios/{runner,definitions,expected}
  fixtures/{stripe,shippo,plaid,zendesk,mock_support,keycloak}
  tests/{integration,end-to-end}
  scripts/{development,deployment,operations}
  .github/{workflows,ISSUE_TEMPLATE}
)

files=(
  README.md
  ARCHITECTURE.md
  DEVELOPMENT.md
  DEPLOYMENT.md
  OPERATIONS.md
  SECURITY.md
  CONTRIBUTING.md
  CHANGELOG.md
  LICENSE
  Makefile
  .gitignore
  .dockerignore
  .editorconfig
  .env.example
  compose.yaml
  compose.override.yaml
  pyproject.toml
  go.work

  services/webhook-gateway/go.mod
  services/webhook-gateway/Dockerfile
  services/webhook-gateway/.env.example
  services/webhook-gateway/cmd/gateway/main.go

  services/integration-api/pyproject.toml
  services/integration-api/Dockerfile
  services/integration-api/.env.example
  services/integration-api/src/accord_api/__init__.py
  services/integration-api/src/accord_api/main.py

  services/integration-worker/pyproject.toml
  services/integration-worker/Dockerfile
  services/integration-worker/.env.example
  services/integration-worker/src/accord_worker/__init__.py
  services/integration-worker/src/accord_worker/main.py

  packages/python/accord-core/pyproject.toml
  packages/python/accord-core/README.md
  packages/python/accord-core/src/accord_core/__init__.py

  packages/python/accord-integrations/pyproject.toml
  packages/python/accord-integrations/README.md
  packages/python/accord-integrations/src/accord_integrations/__init__.py

  database/postgresql/README.md
  database/postgresql/migrations/0001_initial.sql
  database/postgresql/seeds/local.sql
  database/clickhouse/README.md
  database/clickhouse/migrations/0001_initial.sql

  etl/pyproject.toml
  etl/Dockerfile
  etl/README.md
  etl/.env.example
  etl/src/accord_etl/__init__.py
  etl/src/accord_etl/main.py

  infrastructure/docker/README.md
  infrastructure/traefik/traefik.yaml
  infrastructure/traefik/dynamic/routes.yaml
  infrastructure/postgresql/postgresql.conf
  infrastructure/clickhouse/config.xml
  infrastructure/clickhouse/users.xml
  infrastructure/keycloak/realms/accord-realm.json
  infrastructure/prometheus/prometheus.yaml
  infrastructure/prometheus/rules/alerts.yaml
  infrastructure/grafana/provisioning/dashboards/dashboards.yaml
  infrastructure/grafana/provisioning/datasources/datasources.yaml
  infrastructure/grafana/dashboards/integration-overview.json
  infrastructure/loki/loki.yaml

  docs/adr/README.md
  docs/adr/0001-go-webhook-ingestion.md
  docs/adr/0002-postgresql-source-of-truth.md
  docs/adr/0003-clickhouse-analytics.md
  docs/adr/0004-transactional-outbox.md
  docs/adr/0005-idempotency-strategy.md
  docs/adr/0006-reconciliation-strategy.md
  docs/adr/0007-deferred-message-broker.md
  docs/adr/0008-keycloak-authentication.md
  docs/api/README.md
  docs/integrations/README.md
  docs/integrations/stripe.md
  docs/integrations/shippo.md
  docs/integrations/plaid.md
  docs/integrations/support.md
  docs/runbooks/provider-outage.md
  docs/runbooks/retry-operations.md
  docs/runbooks/event-replay.md
  docs/runbooks/reconciliation.md
  docs/runbooks/etl-recovery.md

  scenarios/README.md
  scenarios/runner/__init__.py
  scenarios/runner/main.py
  fixtures/README.md
  tests/integration/README.md
  tests/end-to-end/README.md

  scripts/development/setup.sh
  scripts/development/reset-local.sh
  scripts/deployment/deploy.sh
  scripts/operations/replay-events.sh
  scripts/operations/reconcile.sh

  .github/workflows/python.yaml
  .github/workflows/go.yaml
  .github/workflows/integration-tests.yaml
  .github/workflows/container-builds.yaml
  .github/workflows/security.yaml
  .github/workflows/deploy.yaml
  .github/pull_request_template.md
  .github/CODEOWNERS
  .github/ISSUE_TEMPLATE/bug_report.md
  .github/ISSUE_TEMPLATE/feature_request.md
)

for scenario in \
  happy-path \
  duplicate-webhook \
  payment-failure \
  shipping-failure \
  payout-mismatch \
  refund-after-shipping \
  provider-timeout \
  reconciliation-success \
  reconciliation-failure
do
  files+=(
    "scenarios/definitions/$scenario.yaml"
    "scenarios/expected/$scenario.json"
  )
done

for provider in stripe shippo plaid zendesk mock_support keycloak; do
  files+=("fixtures/$provider/sample.json")
done

mkdir -p -- "${directories[@]}"

for file in "${files[@]}"; do
  mkdir -p -- "$(dirname -- "$file")"
  (set -o noclobber; : > "$file")
done

# Empty .gitkeep files allow Git to track otherwise empty directories.
while IFS= read -r -d '' directory; do
  : > "$directory/.gitkeep"
done < <(find . -type d -empty -print0)

printf 'Accord scaffold created in %s\n' "$PWD"
