# Discordo &middot; [![discord](https://img.shields.io/discord/1297292231299956788?color=5865F2&logo=discord&logoColor=white)](https://discord.com/invite/VzF9UFn2aB) [![ci](https://github.com/ayn2op/discordo/actions/workflows/ci.yml/badge.svg)](https://github.com/ayn2op/discordo/actions/workflows/ci.yml) [![Go Report Card](https://goreportcard.com/badge/github.com/ayn2op/discordo)](https://goreportcard.com/report/github.com/ayn2op/discordo) [![license](https://img.shields.io/github/license/ayn2op/discordo?logo=github)](https://github.com/ayn2op/discordo/blob/master/LICENSE)

Discordo is a lightweight, secure, and feature-rich Discord terminal client. Heavily work-in-progress, expect breaking changes.

![Preview](.github/preview.png)

## Features

- Lightweight
- Configurable
- Mouse & clipboard support
- Attachments
- Notifications
- 2-Factor & QR code authentication
- Discord-flavored markdown

## Installation

### Prebuilt binaries

You can download and install a [prebuilt binary here](https://nightly.link/ayn2op/discordo/workflows/ci/main) for Windows, macOS, or Linux.

### Package managers

- Arch Linux: `yay -S discordo-git`
- Gentoo (available on the guru repos as a live ebuild): `emerge net-im/discordo`
- FreeBSD: `pkg install discordo` or via the ports system `make -C /usr/ports/net-im/discordo install clean`.
- Nix (NixOS, home-manager)
  - Downstream nixpkgs installation: Add `pkgs.discordo` to `environment.systemPackages` or `home.packages`.
  <!-- Temporary until downstream home-manager module --> 
  - Upstream flake installation: Add `inputs.discordo.url = "github:ayn2op/discordo"`. Install using `inputs.discordo.homeModules.default` (`.enable, .package, .settings TOML`).
- Windows (Scoop):

```sh
scoop bucket add vvxrtues https://github.com/vvirtues/bucket
scoop install discordo
```

### Building from source

```bash
git clone https://github.com/ayn2op/discordo
cd discordo
go build .
```

### Wayland clipboard support

`x11-dev` is required for X11 clipboard compatibility:

- Ubuntu: `apt install xwayland`
- Arch Linux: `pacman -S xorg-xwayland`

## Usage

1. Run the `discordo` executable with no arguments.

> If you are logging in using an authentication token, provide the `token` command-line flag to the executable (eg: `--token "OTI2MDU5NTQxNDE2Nzc5ODA2.Yc2KKA.2iZ-5JxgxG-9Ub8GHzBSn-NJjNg"`). The token is stored securely in the default OS-specific keyring. If you are logged in through a browser session, you can find the token in the **Network** tab by inspecting the value of the **Authorization** header in an XHR request.

2. Enter your email and password and click on the "Login" button to continue.

If you see empty windows after logging in, try clicking on the first item in the left pane. This will expand your direct message chats. If you don't see the servers, group them into a single folder. To do this in the regular Discord app, drag and drop one server onto another.

## Configuration

The configuration file allows you to configure and customize the behavior, keybindings, and theme of the application.

- Unix: `$XDG_CONFIG_HOME/discordo/config.toml` or `$HOME/.config/discordo/config.toml`
- Darwin: `$HOME/Library/Application Support/discordo/config.toml`
- Windows: `%AppData%/discordo/config.toml`

Discordo uses the default configuration if a configuration file is not found in the aforementioned path; however, the default configuration file is not written to the path. [The default configuration can be found here](./internal/config/config.toml).

## FAQ

### Manually adding token to keyring

Do this if you get the error:

> failed to get token from keyring: secret not found in keyring

#### Windows

Run the following command in a terminal window. Replace `YOUR_DISCORD_TOKEN` with your authentication token.

```sh
cmdkey /add:discordo /user:token /pass:YOUR_DISCORD_TOKEN
```

#### MacOS

Run the following command in a terminal window. Replace `YOUR_DISCORD_TOKEN` with your authentication token.

```sh
security add-generic-password -s discordo -a token -w "YOUR_DISCORD_TOKEN"
```

#### Linux

1. Start the keyring daemon.

```sh
eval $(gnome-keyring-daemon --start)
export $(gnome-keyring-daemon --start)
```

2. Create the `login` keyring if it does not exist already. See [GNOME/Keyring](https://wiki.archlinux.org/title/GNOME/Keyring) for more information.

3. Run the following command to create the `token` entry.

```sh
secret-tool store --label="Discord Token" service discordo username token
```

4. When it prompts for the password, paste your token, and hit enter to confirm.

> [!IMPORTANT]
> Automated user accounts or "self-bots" are against Discord's Terms of Service. I am not responsible for any loss caused by using "self-bots" or Discordo.
