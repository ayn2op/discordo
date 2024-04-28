self: { options, config, lib, pkgs, ... }: {
  options.programs.discordo = {
    enable = lib.mkEnableOption "discordo";
    package = lib.mkOption {
      type = lib.types.package;
      default = self.packages.${pkgs.system}.default;
      description = "The discordo package to use.";
    };

    settings = lib.mkOption {
      type = pkgs.formats.toml;
      default = { };
    };
  };
  config = lib.mkIf config.discordo.enable {
    home.packages = [ config.discordo.package ];
    programs.discordo.settings = import ./default-config.nix;
    xdg.configFile."discordo/config.toml" = options.discordo.settings.type.generate;
  };
}
