#!/usr/bin/env bash

set -Eeuo pipefail
trap cleanup SIGINT SIGTERM ERR EXIT

script_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")" &>/dev/null && pwd -P)

usage() {
  cat <<EOF
Usage: $(basename "${BASH_SOURCE[0]}") [options]

This script creates a code bundle for deployment to Elastic Beanstalk
by coping all files in the following paths to a new ZIP archive:
- aws/bundle-include/*
- api/*
Additionally handles configuring the TLS/SSL certificate generation script.

Available options:

-h, --help      Print this help and exit
-v, --verbose   Print script debug info
-s, --staging   Uses the LetsEncrypt staging environment
-d, --domains   Comma-separated list of domains to configure TLS/SSL for
-e, --email     Email to associate with LetsEncrypt account
                and send account-related notifications to
EOF
  exit
}

cleanup() {
  trap - SIGINT SIGTERM ERR EXIT
  # script cleanup here

  # Remove the working directory
  if [[ -n ${WORK_DIR+x} ]]; then
    rm -rf "$WORK_DIR"
  fi
}

setup_colors() {
  if [[ -t 2 ]] && [[ -z "${NO_COLOR-}" ]] && [[ "${TERM-}" != "dumb" ]]; then
    NOFORMAT='\033[0m' RED='\033[0;31m' GREEN='\033[0;32m' ORANGE='\033[0;33m' BLUE='\033[0;34m' PURPLE='\033[0;35m' CYAN='\033[0;36m' YELLOW='\033[1;33m'
  else
    NOFORMAT='' RED='' GREEN='' ORANGE='' BLUE='' PURPLE='' CYAN='' YELLOW=''
  fi
}

msg() {
  echo >&2 -e "${1-}"
}

die() {
  local msg=$1
  local code=${2-1} # default exit status 1
  msg "${RED}$msg${NOFORMAT}"
  exit "$code"
}

parse_params() {
  # default values of variables set from params
  staging=0
  domains=''
  email=''

  while :; do
    case "${1-}" in
    -h | --help) usage ;;
    -v | --verbose) set -x ;;
    --no-color) NO_COLOR=1 ;;
    -s | --staging) staging=1 ;;
    -d | --domains)
      domains="${2-}"
      shift
      ;;
    -e | --email)
      email="${2-}"
      shift
      ;;
    -?*) die "Unknown option: $1" ;;
    *) break ;;
    esac
    shift
  done

  # check required params and arguments
  [[ -z "${domains-}" ]] && die "Missing required parameter: domains"
  [[ -z "${email-}" ]] && die "Missing required parameter: param"

  return 0
}

list_files() {
  pushd "$1" >/dev/null
  unset a i
  i=0
  while IFS= read -r -d $'\0' file; do
    a[i++]="  ${PURPLE}${file#./}${NOFORMAT}"
  done < <(find . \( -type f -o -type p -o -type l -o -type s \) -print0 | LC_COLLATE=C sort -z)
  ( IFS=$'\n'; msg "${a[*]}" )
  popd >/dev/null
}

setup_colors
parse_params "$@"

# Create a temporary working directory
WORK_DIR=$(mktemp -d)
if [[ ! "$WORK_DIR" || ! -d "$WORK_DIR" ]]; then
  die "Could not create temp working directory"
fi

msg "${BLUE}Using '$WORK_DIR' as a temp working directory${NOFORMAT}"
cp -r "$script_dir"/../api/. "$WORK_DIR"
cp -r "$script_dir"/../aws/bundle-include/. "$WORK_DIR"

# Delete the binary file if it existed in the api/ folder
rm -f "$WORK_DIR"/klemis-kitchen-api

# Template the certificate script
cert_script_path=".platform/hooks/postdeploy/010_get_certificate.sh"
cert_script_variables='$tls_domains:$tls_email:$tls_staging'
export tls_domains="${domains}" tls_email="${email}"
if [[ "$staging" == 1 ]]; then
  export tls_staging="--staging"
else
  export tls_staging=""
fi
cert_script_contents="$(envsubst "$cert_script_variables" < "$WORK_DIR"/"$cert_script_path")"
echo "$cert_script_contents" > "$WORK_DIR"/"$cert_script_path"

msg "${PURPLE}Archive contents:${NOFORMAT}"
list_files "$WORK_DIR"

# Delete the existing archive
rm -f "$script_dir"/../aws-bundle.zip

# Create the archive
pushd "$WORK_DIR" >/dev/null
zip -r "$script_dir"/../aws-bundle.zip . >/dev/null
popd >/dev/null

bundle_path="$(realpath --relative-to="$(pwd)" "$script_dir"/../aws-bundle.zip)"
msg "${GREEN}Generated bundle at ${bundle_path}${NOFORMAT}"
