{pkgs, ...}: {
  default = pkgs.mkShell {
    packages = with pkgs; [
      ## golang
      delve
      go-outline
      go
      golangci-lint
      golangci-lint-langserver
      gopkgs
      gopls
      gotools
    ];
  };
}
