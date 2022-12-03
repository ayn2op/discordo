{ 
  description = "A lightweight, secure, and feature-rich Discord terminal client.";

  inputs.nixpkgs.url = "nixpkgs/nixos-unstable";

  outputs = { self, nixpkgs, ... }: let 
    version = builtins.substring 0 8 self.lastModifiedDate;

    supportedSystems = [ "x86_64-linux" ];

    forAllSystems = nixpkgs.lib.genAttrs supportedSystems;
    nixpkgsFor = forAllSystems (system: import nixpkgs { inherit system; });

  in {

    packages = forAllSystems (system:
      let pkgs = nixpkgsFor.${system};
      in rec {
        default = discordio;

        discordio = pkgs.buildGoModule {
          pname = "discordio";
          inherit version;
          src = ./.;
          vendorSha256 = "sha256-J6J7Tm/GN7Ftxlt10DG9+9LhB8VnLMgEbr4bq5LeEjY=";
        };
      });

    defaultPackage = forAllSystems (system: self.packages.${system}.discordio);


    #defaultApp = forAllSystems (system: {
    #  type = "app";
    #  program = "${self.packages.${system}.discordio}/bin/discordio";
    #});

  };
}
  

