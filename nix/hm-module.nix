self: { options, config, lib, pkgs, ... }:
let
  settingsFormat = pkgs.formats.toml { };
in
{
  options.programs.discordo = {
    enable = lib.mkEnableOption "discordo";
    package = lib.mkOption {
      type = lib.types.package;
      default = self.packages.${pkgs.system}.default;
      description = "The discordo package to use.";
    };

    settings = lib.mkOption {
      type = settingsFormat.type;
      description = ''
        Configuration for discordo.
        See https://github.com/ayn2op/discordo?tab=readme-ov-file#configuration 
        for available options and default values.
      '';
      default = { };
    };

    tokenCommand = lib.mkOption {
      type = with lib.types; nullOr str;
      description = ''
        If not null, wraps discordo with -token set to the output of tokenCommand.
        Useful if you want token authentication instead of email & password.

        Note that since the wrapper is made using writeShellApplication, this command will
        will be checked by ShellCheck.
      '';
      default = null;
      defaultText = ''
        # password-store method
        "''${lib.getExe pass} discordo-token"

        # sops method
        cat /run/secrets/discordo-token
      '';
    };
  };
  config = lib.mkIf config.programs.discordo.enable {
    home.packages = with config.programs.discordo; [
      (
        if tokenCommand == null then package
        else
          pkgs.writeShellApplication {
            name = "discordo";
            runtimeInputs = [ package ];
            text = ''
              discordo -token "$(${tokenCommand})" "$@"
            '';
          }
      )
    ];

    xdg.configFile."discordo/config.toml".source = settingsFormat.generate
      "discordo-config.toml"
      config.programs.discordo.settings;
  };
}
