#!/bin/bash

export CONDUIT_PROJECT_ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

conduit-dev() {
    (cd "$CONDUIT_PROJECT_ROOT" && go run ./cmd/conduit-dev "$@")
}

conduit() {
    (cd "$CONDUIT_PROJECT_ROOT" && go run ./cmd/conduit "$@")
}

export -f conduit-dev
export -f conduit

echo "🚀 Conduit development environment ready!"
echo ""
echo "Development commands:"
echo "  conduit-dev up      - Start infrastructure (postgres, redis)"
echo "  conduit-dev down    - Stop infrastructure"
echo "  conduit-dev status  - Show service status"
echo "  conduit-dev serve   - Start full dev environment"
echo "  conduit-dev logs    - Tail service logs"
echo ""
echo "Production commands:"
echo "  conduit serve       - Run conduit server"
echo "  conduit migrate     - Database migrations"
echo ""
echo "Get started:"
echo "  conduit-dev up"
echo ""
