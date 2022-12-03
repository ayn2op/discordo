{
  description = "A lightweight, secure, and feature-rich Discord terminal client.";

  inputs.nixpkgs.url = "nixpkgs/nixos-unstable";

  outputs = { self, nixpkgs, ... }: let 
    system = "x86-64-linux"; 
    pname = "discordio";
  in {

    packages = { 
      $(system).$(pname) = nixpkgs.buildGoModule {
        inherit pname;
        version = "1.0.0";
        src ./.;
        vendorSha256 = pkgs.lib.fakeSha256;
      };
    };

    defaultPackage = self.packages.$(system).$(pname);

    defaultApp = {
      type "app";
      program = "${self.packages.$(system).$(pname)}/bin/discordio";
    };
  };
}
  

