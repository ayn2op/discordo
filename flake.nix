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
        default = pkgs.buildGo122Module {
          pname = "discordo";
          version = builtins.substring 0 8 self.lastModifiedDate;
          src = ./.;
          vendorHash = "sha256-hSrGN3NHPpp5601l4KcmNHVYOGWfLjFeWWr9g11nM3I=";
        };
      });
    };
}
  

