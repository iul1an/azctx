# Maintainer: Iulian Mandache <25257851+iul1an@users.noreply.github.com>

pkgname=azctx
pkgver=1.4.2
pkgrel=1
pkgdesc="Per-shell isolated Azure subscription contexts"
arch=('x86_64' 'aarch64')
url="https://github.com/iul1an/azctx"
license=('MIT')
depends=('azure-cli')
makedepends=('go')
source=("$pkgname-$pkgver.tar.gz::$url/archive/refs/tags/v$pkgver.tar.gz")

# make updatesums
sha256sums=('0bbc09b4481658593e279e6acb4f00391479558d533c1747a92546ff6ed9e296')

prepare() {
  cd "$pkgname-$pkgver"
  export GOPATH="$srcdir"
  go mod download -modcacherw
}

build() {
  cd "$pkgname-$pkgver"
  export CGO_CPPFLAGS="${CPPFLAGS}"
  export CGO_CFLAGS="${CFLAGS}"
  export CGO_CXXFLAGS="${CXXFLAGS}"
  export CGO_LDFLAGS="${LDFLAGS}"
  export GOPATH="$srcdir"
  export GOFLAGS="-buildmode=pie -trimpath -mod=readonly -modcacherw"
  go build -ldflags "-linkmode=external -X main.version=v$pkgver" -o azctx .
}

check() {
  cd "$pkgname-$pkgver"
  export GOPATH="$srcdir"
  export GOFLAGS="-mod=readonly -modcacherw"
  go test ./...
}

package() {
  cd "$pkgname-$pkgver"
  install -Dm755 azctx "$pkgdir/usr/bin/azctx"
  install -Dm644 LICENSE "$pkgdir/usr/share/licenses/$pkgname/LICENSE"
  ./azctx completion bash | install -Dm644 /dev/stdin "$pkgdir/usr/share/bash-completion/completions/azctx"
  ./azctx completion zsh | install -Dm644 /dev/stdin "$pkgdir/usr/share/zsh/site-functions/_azctx"
  ./azctx completion fish | install -Dm644 /dev/stdin "$pkgdir/usr/share/fish/vendor_completions.d/azctx.fish"
}
