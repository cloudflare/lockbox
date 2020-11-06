{ config, lib, pkgs, controller, ... }:

pkgs.dockerTools.buildLayeredImage {
  name = "lockbox-controller";
  config.Entrypoint = [ "${controller}/bin/lockbox-controller" ];
}
