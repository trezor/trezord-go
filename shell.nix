with import <nixpkgs> {};

stdenv.mkDerivation {
  name = "trezord-go-env";
  buildInputs = [
    fpm
    gcc
    gnupg
    go
    nsis
    osslsigncode
    pkgsCross.aarch64-multiplatform.buildPackages.gcc
    pkgsCross.mingwW64.buildPackages.gcc
    rpm
  ];
}
