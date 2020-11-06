{ pkgs ? import (builtins.fetchTarball {
  name = "nixos-2009-2020-11-06";
  url = "https://github.com/NixOS/nixpkgs/archive/d105075a1fd870b1d1617a6008cb38b443e65433.tar.gz";
  sha256 = "1jcs44wn0s6mlf2jps25bvcai1rij9b2dil6zcj8zqwf2i8xmqjh";
}) {} }:

let d = pkgs.callPackage ./hack/derivation.nix {};

in {
  locket = d.locket;
  lockbox = import ./hack/docker.nix {
    pkgs = pkgs;
    lib = pkgs.lib;
    config = pkgs.config;
    controller = d.lockbox;
  };
}
