self: { options, config, lib, pkgs, ... }:
let
  cfg = config.programs.discordo;
  settingsFormat = pkgs.formats.toml { };
in
{
  options.programs.discordo = {
    enable = lib.mkEnableOption "discordo";
    package = lib.mkPackageOption self.packages.${pkgs.system} "discordo" { };
    settings = lib.mkOption {
      type = settingsFormat.type;
      description = ''
        Configuration for discordo.
        See https://github.com/ayn2op/discordo?tab=readme-ov-file#configuration 
        for available options and default values.
      '';
      default = { };
    };
  };
  config = lib.mkIf cfg.enable {
    home.packages = [ cfg.package ];
    xdg.configFile."discordo/config.toml".source = settingsFormat.generate
      "discordo-config.toml"
      cfg.settings;
  };
}
      

