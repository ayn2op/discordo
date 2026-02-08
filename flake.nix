{
  inputs = {
    flake-parts.url = "github:hercules-ci/flake-parts";
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
    systems.url = "github:nix-systems/x86_64-linux";
  };

  outputs = {
    flake-parts,
    systems,
    ...
  } @ inputs:
    flake-parts.lib.mkFlake {inherit inputs;} {
      systems = import systems;

      perSystem = {pkgs, ...}: {
        packages = rec {
          default = discordo;
          discordo = pkgs.callPackage ./nix/package.nix {};
        };
        devShells = import ./nix/shell.nix {inherit pkgs;};
        formatter = pkgs.alejandra;
      };
    };
}
