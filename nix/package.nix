{
  lib,
  fetchFromGitHub,
  buildGoModule,
  makeWrapper,
  xsel,
  wl-clipboard,
  stdenv,
}: let
  pname = "discordo";
  version = "0-unstable-2026-01-23";
in
  buildGoModule {
    inherit pname version;

    src = ../.;

    vendorHash = "sha256-/HUBzSq4VIep0Q/UYAGCUIbzEmlRSHZRjGJuJi7OFn4=";
    ldflags = [
      "-s"
      "-X main.version=${version}"
    ];

    env.CGO_ENABLED = 0;

    nativeBuildInputs = lib.optionals stdenv.hostPlatform.isLinux [
      makeWrapper
    ];

    postInstall = lib.optionalString stdenv.hostPlatform.isLinux ''
      wrapProgram $out/bin/discordo \
        --prefix PATH : ${
        lib.makeBinPath [
          xsel
          wl-clipboard
        ]
      }
    '';

    meta = {
      description = "Lightweight, secure, and feature-rich Discord terminal client";
      homepage = "https://github.com/ayn2op/discordo";
      license = lib.licenses.gpl3Plus;
      mainProgram = "discordo";
    };
  }
