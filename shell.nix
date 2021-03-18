{ pkgs ? import ./hack/nixpkgs.nix { }, ... }:

with pkgs;

mkShell { buildInputs = [ go goimports ]; }
