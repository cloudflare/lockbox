{ config, lib, pkgs, ... }:

let
  srcs = lib.sourceFilesBySuffices ../. [ ".go" ".mod" ".sum" ];
  v256 = "1p1v43zxm4g2fra20l5p704y0p7rmgqiyffx392w5g5r22hgxjyg";

in {
  lockbox = pkgs.buildGo117Module rec {
    pname = "lockbox";
    version = "0.5.0";

    src = srcs;
    ldflags = [ "-X main.version=${version}" ];

    vendorSha256 = v256;

    subPackages = [ "cmd/lockbox-controller" ];
  };
  locket = pkgs.buildGo117Module rec {
    pname = "locket";
    version = "0.5.0";

    src = srcs;
    ldflags = [ "-X main.version=${version}" ];

    vendorSha256 = v256;

    subPackages = [ "cmd/locket" ];
  };
}
