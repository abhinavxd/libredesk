#!/bin/sh
set -eu
umask 077

mode=${1:-local}
case "$mode" in
  local) root_url=http://localhost:9000 ;;
  production)
    root_url=${2:-}
    case "$root_url" in
      https://?*) ;;
      *) printf '%s\n' 'Production needs your HTTPS address: sh setup.sh production https://support.example.com' >&2; exit 1 ;;
    esac
    ;;
  *) printf '%s\n' 'Usage: sh setup.sh local | production https://support.example.com' >&2; exit 1 ;;
esac

for command in curl openssl docker; do
  command -v "$command" >/dev/null 2>&1 || { printf 'Install %s before running setup.\n' "$command" >&2; exit 1; }
done
docker compose version >/dev/null

if [ -e config.toml ] || [ -e .env ]; then
  if [ ! -f config.toml ] || [ ! -f .env ] || [ ! -f docker-compose.yml ]; then
    printf '%s\n' 'Existing setup files are incomplete. Restore config.toml, .env, and docker-compose.yml before continuing. Nothing was overwritten.' >&2
    exit 1
  fi
  root_url=$(sed -n 's/^LIBREDESK_APP__ROOT_URL=//p' .env)
  root_url=${root_url:-http://localhost:9000}
  printf '%s\n' 'Existing configuration and secrets were preserved.'
else
  for path in docker-compose.yml uploads; do
    if [ -e "$path" ]; then
      printf '%s already exists. Run setup in an empty directory. Nothing was overwritten.\n' "$path" >&2
      exit 1
    fi
  done
  release=${LIBREDESK_RELEASE:-}
  if [ -z "$release" ]; then
    release_url=$(curl -fsSL -o /dev/null -w '%{url_effective}' https://github.com/abhinavxd/libredesk/releases/latest)
    release=${release_url##*/}
  fi
  case "$release" in
    v[0-9]*) ;;
    *) printf '%s\n' 'Could not identify a release. Set LIBREDESK_RELEASE to a published release tag.' >&2; exit 1 ;;
  esac
  case "$release" in
    *[!a-zA-Z0-9._-]*) printf '%s\n' 'Release tag contains invalid characters.' >&2; exit 1 ;;
  esac
  staging=$(mktemp -d .libredesk-setup.XXXXXX)
  trap 'rm -rf "$staging"' EXIT HUP INT TERM
  source_url="https://raw.githubusercontent.com/abhinavxd/libredesk/$release"
  curl -fsSL "$source_url/config.sample.toml" -o "$staging/config.sample.toml"
  curl -fsSL "$source_url/docker-compose.yml" -o "$staging/docker-compose.yml"
  encryption_key=$(openssl rand -hex 16)
  database_password=$(openssl rand -hex 24)
  sed -e "s/your-32-char-random-string-here!/$encryption_key/" \
    -e "s/password = \"libredesk\"/password = \"$database_password\"/" \
    -e 's/env = "dev"/env = "prod"/' \
    -e 's/log_level = "debug"/log_level = "info"/' \
    "$staging/config.sample.toml" > "$staging/config.toml"
  {
    printf 'LIBREDESK_IMAGE=libredesk/libredesk:%s\n' "$release"
    printf 'POSTGRES_PASSWORD=%s\n' "$database_password"
    printf '%s\n' 'LIBREDESK_HTTP_BIND=127.0.0.1' 'POSTGRES_PUBLISHED_PORT=0' 'REDIS_PUBLISHED_PORT=0'
    printf 'LIBREDESK_APP__ROOT_URL=%s\n' "$root_url"
  } > "$staging/.env"
  mkdir uploads
  mv "$staging/config.toml" config.toml
  mv "$staging/.env" .env
  mv "$staging/docker-compose.yml" docker-compose.yml
  printf 'Configuration created for %s. Back up config.toml, .env, and uploads with your database.\n' "$root_url"
fi

unset POSTGRES_USER POSTGRES_PASSWORD POSTGRES_DB LIBREDESK_IMAGE LIBREDESK_HTTP_BIND
unset POSTGRES_PUBLISHED_PORT REDIS_PUBLISHED_PORT LIBREDESK_APP__ROOT_URL
docker compose --env-file .env -f docker-compose.yml up -d --wait
docker compose --env-file .env -f docker-compose.yml logs --tail=30 app
printf 'Open %s. Existing accounts and passwords are unchanged.\n' "$root_url"
if [ -n "${LIBREDESK_SYSTEM_USER_PASSWORD:-}" ]; then
  printf '%s\n' 'For a new installation, sign in as System with the password you supplied.'
else
  printf 'If you have not created an admin account, open %s/setup and use the temporary code in the logs.\n' "$root_url"
fi
if [ "$mode" = production ]; then
  printf '%s\n' 'Configure your HTTPS reverse proxy before opening setup. See INSTALL.md in the release archive.'
fi
