{ config, lib, pkgs, ... }:

let
  srcs = lib.sourceFilesBySuffices ../. [ ".go" ".mod" ".sum" ];
  v256 = "sha256-ztGHplIcejWeFSJS3Ykd702L8/RJcmQ9jstJeVHtL38=";

in {
  lockbox = pkgs.buildGoModule rec {
    pname = "lockbox";
    version = "0.6.0";

    src = srcs;
    ldflags = [ "-X main.version=${version}" ];

    vendorSha256 = v256;

    subPackages = [ "cmd/lockbox-controller" ];
  };
  locket = pkgs.buildGo117Module rec {
    pname = "locket";
    version = "0.6.0";

    src = srcs;
    ldflags = [ "-X main.version=${version}" ];

    vendorSha256 = v256;

    subPackages = [ "cmd/locket" ];
  };
}
