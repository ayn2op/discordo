{
  description = "A lightweight, secure, and feature-rich Discord terminal client.";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs = { self, nixpkgs, ... }:
    let

      supportedSystems = [
        "x86_64-linux"
        "aarch64-linux"
        "x86_64-darwin"
        "aarch64-darwin"
      ];

      forSupportedSystems = f:
        builtins.listToAttrs (builtins.map
          (system:
            {
              name = system;
              value = f system (import nixpkgs { inherit system; });
            })
          supportedSystems);

    in
    {
      packages = forSupportedSystems (system: pkgs: {
        default = pkgs.callPackage ./nix/package.nix { };
      });
      homeManagerModules.default = import ./nix/hm-module.nix self;
    };
}
  

