{ 
  description = "A lightweight, secure, and feature-rich Discord terminal client.";

  inputs.nixpkgs.url = "nixpkgs/nixos-23.05";

  outputs = { self, nixpkgs, ... }: let 

    supportedSystems = [ 
      "x86_64-linux" 
      "aarch64-linux"
      "x86_64-darwin"
    ];

    forSupportedSystems = f:
      builtins.listToAttrs (builtins.map 
        (system: 
          { name = system; value = f system (import nixpkgs { inherit system; }); 
        }) 
        supportedSystems);
    
  in {
    packages = forSupportedSystems (system: pkgs: 
      let pkg = pkgs.buildGoModule {
        pname = "discordo";
        version = builtins.substring 0 8 self.lastModifiedDate;
        src = ./.;
        vendorSha256 = "sha256-5Y+SP374Bd8F2ABKEKRhTcGNhsFM77N5oC5wRN6AzKk=";
      };
    in {
      default = pkg;
      discordo = pkg;
    });
  };
}
  

