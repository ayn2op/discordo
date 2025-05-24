{ buildGoModule
, lib
, makeWrapper
, xsel
, wl-clipboard

, xorgClipboardSupport ? true
, waylandClipboardSupport ? true
}:
let
  anyClipboardSupport = xorgClipboardSupport || waylandClipboardSupport;
in
buildGoModule {
  pname = "discordo";
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

  nativeBuildInputs = lib.optional anyClipboardSupport makeWrapper;

  postInstall = lib.optionalString xorgClipboardSupport ''
    wrapProgram $out/bin/discordo \
      --prefix PATH : ${lib.makeBinPath [ xsel ]}
  '' + lib.optionalString waylandClipboardSupport ''
    wrapProgram $out/bin/discordo \
      --prefix PATH : ${lib.makeBinPath [ wl-clipboard ]}
  '';

  meta = {
    description = "A lightweight, secure, and feature-rich Discord terminal client";
    homepage = "https://github.com/ayn2op/discordo";
    license = lib.licenses.gpl3;
    maintainers = [ lib.maintainers.arian-d ];
    mainProgram = "discordo";
  };
}

