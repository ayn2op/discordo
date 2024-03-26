# Discordo &middot; [![ci](https://github.com/ayn2op/discordo/actions/workflows/ci.yml/badge.svg)](https://github.com/ayn2op/discordo/actions/workflows/ci.yml) [![Go Report Card](https://goreportcard.com/badge/github.com/ayn2op/discordo)](https://goreportcard.com/report/github.com/ayn2op/discordo) [![license](https://img.shields.io/github/license/ayn2op/discordo?logo=github)](https://github.com/ayn2op/discordo/blob/master/LICENSE) [![discord](https://img.shields.io/discord/1069288288034242642)](https://discord.gg/Gx48RRmKvx)

Discordo is a lightweight, secure, and feature-rich Discord terminal client. Heavily work-in-progress, expect breaking changes.

![Preview](.github/preview.png)

- Lightweight
- Secure
- Configurable
- Cross-platform
- Minimalistic
- Feature-rich
  - Mouse & clipboard support
  - 2-Factor authentication
  - Partial [Discord-flavored markdown](https://support.discord.com/hc/en-us/articles/210298617-Markdown-Text-101-Chat-Formatting-Bold-Italic-Underline-)

## Installation

### Prebuilt binaries

You can download and install a [prebuilt binary here](https://nightly.link/ayn2op/discordo/workflows/ci/main) for Windows, macOS, or Linux.

### Package managers

- Arch Linux: `yay -S discordo-git`
- FreeBSD: `pkg install discordo` or via the ports system `make -C /usr/ports/net-im/discordo install clean`.
- NixOS: `nix-shell -p discordo`

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

## Configuration

The configuration file allows you to configure and customize the behavior, keybindings, and theme of the application.

- Unix: `$XDG_CONFIG_HOME/discordo/config.yml` or `$HOME/.config/discordo/config.yml`
- Darwin: `$HOME/Library/Application Support/discordo/config.yml`
- Windows: `%AppData%/discordo/config.yml`

```toml
mouse = true
timestamps = false
timestamps_before_author = false
messages_limit = 50
editor = "default"

[keys.normal]
  insert_mode = "Rune[i]"
  focus_guilds_tree = "Ctrl+G"
  focus_messages_text = "Ctrl+T"
  toggle_guild_tree = "Ctrl+B"

  guilds_tree = {
    select_current = "Enter"
    select_previous = "Rune[k]"
    select_next = "Rune[j]"
    select_first = "Rune[g]"
    select_last = "Rune[G]"
  }

  messages_text = {
    select_previous = "Rune[k]"
    select_next = "Rune[j]"
    select_first = "Rune[g]"
    select_last = "Rune[G]"
    select_reply = "Rune[s]"
    reply = "Rune[r]"
    reply_mention = "Rune[R]"
    delete = "Rune[d]"
    yank = "Rune[y]"
    open = "Rune[o]"
  }

[keys.insert]
  normal_mode = "Esc"
  
  message_input = {
    send = "Enter"
    editor = "Ctrl+E"
  }

[theme]
  border = true
  border_color = "default"
  border_padding = [0, 0, 1, 1]
  title_color = "default"
  background_color = "default"

[theme.guilds_tree]
  auto_expand_folders = true
  graphics = true

[theme.messages_text]
  author_color = "aqua"
  reply_indicator = "╭ "
```

## Documentation

[Here.](./docs)

## Disclaimer

Automated user accounts or "self-bots" are against Discord's Terms of Service. I am not responsible for any loss caused by using "self-bots" or Discordo.
