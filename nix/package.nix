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

  vendorHash = "sha256-gEwTpt/NPN1+YpTBmW8F34UotowrOcA0mfFgBdVFiTA=";
}
