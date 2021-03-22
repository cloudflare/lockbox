{ config, lib, pkgs, ... }:

let
  srcs = lib.sourceFilesBySuffices ../. [ ".go" ".mod" ".sum" ];
  v256 = "0n8k2syjv2b1fd6w3rkq2lm8kdyyj0f8dqprxarvx1pywdznamvf";

in {
  lockbox = pkgs.buildGoModule rec {
    pname = "lockbox";
    version = "0.5.0";

    src = srcs;

    vendorSha256 = v256;

    subPackages = [ "cmd/lockbox-controller" ];
  };
  locket = pkgs.buildGoModule rec {
    pname = "locket";
    version = "0.5.0";

    src = srcs;
    buildFlagsArray = [ "-ldflags=" "-X=main.version=${version}" ];

    vendorSha256 = v256;

    subPackages = [ "cmd/locket" ];
  };
}
