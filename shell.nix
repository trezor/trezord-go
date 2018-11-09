with import <nixpkgs> {};

stdenv.mkDerivation {
  name = "trezord-go-dev";
  buildInputs = [ go ];
}
