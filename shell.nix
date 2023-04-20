{ pkgs ? import ./hack/nixpkgs.nix { }, ... }:

with pkgs;

let
  controller-tools = buildGoModule rec {
    pname = "controller-tools";
    version = "0.11.3";

    src = fetchFromGitHub {
      owner = "kubernetes-sigs";
      repo = "controller-tools";
      rev = "v${version}";
      sha256 = "sha256-F+cGJbxXIHrgn9OcIMh2am9g4PBLkdC037tV/9znPSg=";
    };
    subPackages = [ "cmd/controller-gen" ];

    vendorSha256 = "sha256-nZyDoME5fVqRoAeLADjrQ7i6mVf3ujGN2+BUfrSHck8=";
  };
in pkgs.mkShell {
  nativeBuildInputs = [ go gopls go-tools controller-tools ];

  TEST_ASSET_KUBE_APISERVER = "${kubernetes}/bin/kube-apiserver";
  TEST_ASSET_ETCD = "${etcd}/bin/etcd";
  TEST_ASSET_KUBECTL = "${kubectl}/bin/kubectl";
}
