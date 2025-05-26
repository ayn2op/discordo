{ discordo
, lib
}: discordo.overrideAttrs {
  version = "git";

  src = let fs = lib.fileset; in fs.toSource {
    root = ../.;
    fileset = fs.unions [
      ../go.mod
      ../go.sum
      ../main.go
      ../cmd
      ../internal
    ];
  };

  vendorHash = "sha256-Q9ROPLRP8HSx4P30bSdX30qB2Q1oERz+gZ7Tb23oXbI=";
}
