 { pkgs ? import ./nixpkgs.nix { }, ... }:

 with pkgs;

 let package = callPackage ./derivation.nix { };

 in dockerTools.buildLayeredImage {
   name = "lockbox-controller";
   config = {
     Entrypoint = [ "${package.lockbox}/bin/lockbox-controller" ];
     Env = [ "NIX_SSL_CERT_FILE=${cacert}/etc/ssl/certs/ca-bundle.crt" ];
   };
 }
