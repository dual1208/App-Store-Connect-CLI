#!/bin/sh
set -eu
umask 077

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
output_path=${1:-"$repo_root/asc"}
go_binary=${ASC_GO_BINARY:-go}
required_go_version=go1.26.5
live_tls=${ASC_FILEONLY_LIVE_TLS:-}

case $output_path in
	"")
		echo "output path must not be empty" >&2
		exit 2
		;;
esac

actual_go_version=$($go_binary version | awk '{print $3}')
if [ "$actual_go_version" != "$required_go_version" ]; then
	echo "file-only build requires $required_go_version; found $actual_go_version" >&2
	exit 1
fi

source_goroot=$($go_binary env GOROOT)
replacement="$repo_root/scripts/toolchain/x509_root_darwin_fileonly.go.txt"
replacement_test="$repo_root/scripts/toolchain/x509_root_darwin_fileonly_test.go.txt"
if [ ! -f "$source_goroot/src/crypto/x509/root_darwin.go" ]; then
	echo "expected Darwin x509 source is missing from GOROOT: $source_goroot" >&2
	exit 1
fi
if [ ! -f "$replacement" ]; then
	echo "file-only x509 replacement is missing: $replacement" >&2
	exit 1
fi
if [ ! -f "$replacement_test" ]; then
	echo "file-only x509 test is missing: $replacement_test" >&2
	exit 1
fi

output_dir_input=$(dirname -- "$output_path")
output_name=$(basename -- "$output_path")
if [ ! -d "$output_dir_input" ]; then
	echo "output directory does not exist: $output_dir_input" >&2
	exit 1
fi
output_dir=$(CDPATH= cd -- "$output_dir_input" && pwd)
output_path="$output_dir/$output_name"
if [ -e "$output_path" ] || [ -L "$output_path" ]; then
	echo "refusing to overwrite existing output: $output_path" >&2
	exit 1
fi

temp_parent_input=${TMPDIR:-/tmp}
if [ ! -d "$temp_parent_input" ]; then
	echo "temporary directory does not exist: $temp_parent_input" >&2
	exit 1
fi
temp_parent=$(CDPATH= cd -- "$temp_parent_input" && pwd)
build_root=$(mktemp -d "$temp_parent/asc-file-only-build.XXXXXX")
marker="$build_root/.asc-file-only-build-root"
: >"$marker"
publish_temp=
cleanup() {
	if [ -n "$publish_temp" ]; then
		case $publish_temp in
			"$output_dir"/.asc-file-only-output.*)
				if [ -e "$publish_temp" ] || [ -L "$publish_temp" ]; then
					unlink "$publish_temp"
				fi
				;;
		esac
	fi
	case $build_root in
		"$temp_parent"/asc-file-only-build.*)
			if [ -f "$marker" ]; then
				find "$build_root" -depth -delete
			fi
			;;
	esac
}
trap cleanup EXIT
trap 'exit 129' HUP
trap 'exit 130' INT
trap 'exit 143' TERM

patched_goroot="$build_root/goroot"
mkdir "$patched_goroot"
for source_entry in "$source_goroot"/*; do
	entry_name=$(basename -- "$source_entry")
	if [ "$entry_name" != src ]; then
		ln -s "$source_entry" "$patched_goroot/$entry_name"
	fi
done
mkdir "$patched_goroot/src"
for source_entry in "$source_goroot/src"/*; do
	entry_name=$(basename -- "$source_entry")
	if [ "$entry_name" != crypto ]; then
		ln -s "$source_entry" "$patched_goroot/src/$entry_name"
	fi
done
mkdir "$patched_goroot/src/crypto"
for source_entry in "$source_goroot/src/crypto"/*; do
	entry_name=$(basename -- "$source_entry")
	if [ "$entry_name" != x509 ]; then
		ln -s "$source_entry" "$patched_goroot/src/crypto/$entry_name"
	fi
done
cp -R "$source_goroot/src/crypto/x509" "$patched_goroot/src/crypto/x509"
find "$patched_goroot/src/crypto/x509" -type d -exec chmod u+w {} +
chmod u+w "$patched_goroot/src/crypto/x509/root_darwin.go"
cp "$replacement" "$patched_goroot/src/crypto/x509/root_darwin.go"
cp "$replacement_test" "$patched_goroot/src/crypto/x509/root_darwin_fileonly_test.go"

commit=unknown
build_date=unknown
if command -v git >/dev/null 2>&1 && git -C "$repo_root" rev-parse --verify HEAD >/dev/null 2>&1; then
	commit=$(git -C "$repo_root" rev-parse HEAD)
	build_date=$(git -C "$repo_root" show -s --format=%cI HEAD)
fi

candidate="$build_root/asc"
(
	cd "$repo_root"
	ASC_FILEONLY_LIVE_TLS= \
	CGO_ENABLED=0 \
	GOROOT="$patched_goroot" \
	GOTOOLCHAIN=local \
	GOPROXY=off \
	GOSUMDB=off \
	GONOSUMDB='*' \
	GOPRIVATE='*' \
	GOFLAGS='-mod=readonly' \
	"$patched_goroot/bin/go" test -count=1 ./scripts/toolchain/testdata/fileonlytls

	CGO_ENABLED=0 \
	GOROOT="$patched_goroot" \
	GOTOOLCHAIN=local \
	GOPROXY=off \
	GOSUMDB=off \
	GONOSUMDB='*' \
	GOPRIVATE='*' \
	GOFLAGS='-mod=readonly' \
	"$patched_goroot/bin/go" test -count=1 crypto/x509 -run '^TestFileOnly'

	if [ "$live_tls" = 1 ]; then
		ASC_FILEONLY_LIVE_TLS=1 \
		CGO_ENABLED=0 \
		GOROOT="$patched_goroot" \
		GOTOOLCHAIN=local \
		GOPROXY=off \
		GOSUMDB=off \
		GONOSUMDB='*' \
		GOPRIVATE='*' \
		GOFLAGS='-mod=readonly' \
		"$patched_goroot/bin/go" test -count=1 ./scripts/toolchain/testdata/fileonlytls -run '^TestASCReadOnlyLiveTLS$'
	fi

	CGO_ENABLED=0 \
	GOROOT="$patched_goroot" \
	GOTOOLCHAIN=local \
	GOPROXY=off \
	GOSUMDB=off \
	GONOSUMDB='*' \
	GOPRIVATE='*' \
	GOFLAGS='-mod=readonly' \
	"$patched_goroot/bin/go" build \
		-trimpath \
		-buildvcs=false \
		-ldflags "-s -w -X main.version=clean.5 -X main.commit=$commit -X main.date=$build_date" \
		-o "$candidate" \
		.
)

if otool -L "$candidate" | grep -E 'Security\.framework|CoreFoundation\.framework' >/dev/null; then
	echo "forbidden Apple security framework linkage detected" >&2
	exit 1
fi
if nm -u "$candidate" | grep -E 'Sec(Item|Keychain)|_security_' >/dev/null; then
	echo "forbidden native credential-store symbol detected" >&2
	exit 1
fi
if strings "$candidate" | grep -E 'Security\.framework|SecItem|SecKeychain|Keychain|keyring|native credential store|github\.com/1Password/srp|bitrise-io/go-xcode' >/dev/null; then
	echo "forbidden credential-store or removed-subsystem string detected" >&2
	exit 1
fi

publish_temp=$(mktemp "$output_dir/.asc-file-only-output.XXXXXX")
install -m 0755 "$candidate" "$publish_temp"
ln "$publish_temp" "$output_path"
unlink "$publish_temp"
publish_temp=

echo "built $output_path"
shasum -a 256 "$output_path"
