# Discordo &middot; [![discord](https://img.shields.io/discord/1297292231299956788?color=5865F2&logo=discord&logoColor=white)](https://discord.com/invite/VzF9UFn2aB) [![ci](https://github.com/ayn2op/discordo/actions/workflows/ci.yml/badge.svg)](https://github.com/ayn2op/discordo/actions/workflows/ci.yml) [![Go Report Card](https://goreportcard.com/badge/github.com/ayn2op/discordo)](https://goreportcard.com/report/github.com/ayn2op/discordo) [![license](https://img.shields.io/github/license/ayn2op/discordo?logo=github)](https://github.com/ayn2op/discordo/blob/master/LICENSE)

Discordo is a lightweight, secure, and feature-rich Discord terminal client. Heavily work-in-progress, expect breaking changes.

![Preview](.github/preview.png)

## Features

- Lightweight
- Configurable
- Mouse & clipboard support
- Notifications
- 2-Factor authentication
- [Discord-flavored markdown](https://support.discord.com/hc/en-us/articles/210298617-Markdown-Text-101-Chat-Formatting-Bold-Italic-Underline-)

## Installation

### Prebuilt binaries

You can download and install a [prebuilt binary here](https://nightly.link/ayn2op/discordo/workflows/ci/main) for Windows, macOS, or Linux.

### Package managers

- Arch Linux: `yay -S discordo-git`
- FreeBSD: `pkg install discordo` or via the ports system `make -C /usr/ports/net-im/discordo install clean`.
- NixOS: `nix-shell -p discordo`

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

### Linux clipboard support

- `xclip` or `xsel` for X11 (`apt install xclip`)
- `wl-clipboard` for Wayland (`apt install wl-clipboard`)

## Usage

1. Run the `discordo` executable with no arguments.

> If you are logging in using an authentication token, provide the `token` command-line flag to the executable (eg: `--token "OTI2MDU5NTQxNDE2Nzc5ODA2.Yc2KKA.2iZ-5JxgxG-9Ub8GHzBSn-NJjNg"`). The token is stored securely in the default OS-specific keyring.

2. Enter your email and password and click on the "Login" button to continue.

## Keymaps

### Global

- `Ctrl+G`: Focus Guilds Tree
- `Ctrl+T`: Focus Messages Text
- `Ctrl+P`: Focus Message Input
- `Ctrl+B`: Toggle Guilds Tree (sidebar)
- `Esc`: Reset message selection or close the channel selection popup.
- `Ctrl+C`: Quit the application.
- `Ctrl+D`: Log out and remove the authentication token from keyring (requires re-login upon restart).

#### Navigation

- `k`: Select Previous (any context except input)
- `j`: Select Next  (any context except input)
- `g`: Select First (any context except input)
- `G`: Select Last (any context except input)

### Guilds Tree

- `Enter`: Select the currently highlighted text-based channel or expand a guild or channel.

### Message Text

- `s`: Select the message reference (reply) of the selected channel.
- `p`: Select the pinned message.
- `r`: Reply to the selected message.
- `R`: Reply (with mention) to the selected message.
- `d`: Delete the selected message.
- `y`: Yank (copy) the selected message's content.
- `o`: Open the selected message's attachments in the default browser application.

### Message Input

- `Alt+Enter`: Insert a new line to the current text.
- `Enter`: Send the message.
- `Ctrl+E`: Open message input in your default `$EDITOR`.
- `Esc`: Remove existing text or cancel reply.

## Configuration

The configuration file allows you to configure and customize the behavior, keybindings, and theme of the application.

- Unix: `$XDG_CONFIG_HOME/discordo/config.toml` or `$HOME/.config/discordo/config.toml`
- Darwin: `$HOME/Library/Application Support/discordo/config.toml`
- Windows: `%AppData%/discordo/config.toml`

[The default configuration can be found here](./internal/config/config.go).

## FAQ

### Manually adding token to keyring

Do this if you get the error:

> failed to get token from keyring: secret not found in keyring

#### MacOS

Run the following command in a terminal window with `sudo` to create the `token` entry.

```sh
security add-generic-password -s discordo -a token -w "DISCORD TOKEN HERE"
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
secret-tool store --label="DISCORD TOKEN HERE" service discordo username token
```

4. When it prompts for the password, paste your token, and hit enter to confirm.

## Disclaimer

Automated user accounts or "self-bots" are against Discord's Terms of Service. I am not responsible for any loss caused by using "self-bots" or Discordo.
