with import <nixpkgs> {};

stdenv.mkDerivation {
  name = "onekey-go-dev";
  buildInputs = [ go ];
}
