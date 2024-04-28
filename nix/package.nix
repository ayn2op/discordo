{ buildGo122Module
, lib
, makeWrapper
, xsel
, wl-clipboard
, guiSupport ? true
, xorgClipboardSupport ? guiSupport
, waylandClipboardSupport ? guiSupport
}:
let
  anyClipboardSupport = xorgClipboardSupport || waylandClipboardSupport;
in
buildGo122Module {
  pname = "discordo";
  version = "unstable-2024-04-28";

  src = ./..;
  vendorHash = "sha256-hSrGN3NHPpp5601l4KcmNHVYOGWfLjFeWWr9g11nM3I=";
  # doCheck = false;

  nativeBuildInputs = lib.optional anyClipboardSupport makeWrapper;

  postInstall =
    let
      clipboardPkgs =
        lib.optional xorgClipboardSupport xsel
        ++ lib.optional waylandClipboardSupport wl-clipboard;
    in
    lib.optionalString anyClipboardSupport ''
      wrapProgram $out/bin/discordo \
        --prefix PATH : ${lib.makeBinPath clipboardPkgs}
    '';

  meta = {
    description = "A lightweight, secure, and feature-rich Discord terminal client";
    homepage = "https://github.com/ayn2op/discordo";
    license = lib.licenses.mit;
    maintainers = [ lib.maintainers.arian-d ];
    mainProgram = "discordo";
  };
}

