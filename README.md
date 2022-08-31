# Discordo &middot; [![ci](https://github.com/ayntgl/discordo/actions/workflows/ci.yml/badge.svg)](https://github.com/ayntgl/discordo/actions/workflows/ci.yml) [![Go Report Card](https://goreportcard.com/badge/github.com/ayntgl/discordo)](https://goreportcard.com/report/github.com/ayntgl/discordo) [![license](https://img.shields.io/github/license/ayntgl/discordo?logo=github)](https://github.com/ayntgl/discordo/blob/master/LICENSE)

Discordo is a lightweight, secure, and feature-rich Discord terminal client. Heavily work-in-progress, expect breaking changes.

![Preview](.github/preview.png)

## Table of Contents

- [Features](#features)
- [Installation](#installation)
  - [Prebuilt binaries](#prebuilt-binaries)
  - [Package managers](#package-managers)
  - [Building from source](#building-from-source)
- [Usage](#usage)
  - [Configuration](#configuration)
- [Disclaimer](#disclaimer)

## Features

- Lightweight
- Secure
- Configurable
- Cross-platform
- Minimalistic
- Feature-rich
  - Mouse & clipboard support
  - 2-Factor authentication
  - Desktop notifications
  - Partial [Discord-flavored markdown](https://support.discord.com/hc/en-us/articles/210298617-Markdown-Text-101-Chat-Formatting-Bold-Italic-Underline-)

## Installation

### Prebuilt binaries

You can download and install a [prebuilt binary here](https://nightly.link/ayntgl/discordo/workflows/ci/main) for Windows, macOS, or Linux.

### Package managers

- [Arch Linux](https://aur.archlinux.org/packages/discordo-git/): `yay -S discordo-git` (thanks to [Alyxia Sother](https://github.com/lexisother) for maintaining the AUR package).

### Building from source

```bash
git clone https://github.com/ayntgl/discordo
cd discordo
make build

# optional
sudo mv ./discordo /usr/local/bin
```

### Linux clipboard support

- `xclip` or `xsel` for X11.
  - Ubuntu: `apt install xclip`
  - Arch Linux: `pacman -S xclip`
  - Fedora: `dnf install xclip`
- `wl-clipboard` for Wayland.
  - Ubuntu: `apt install wl-clipboard`
  - Arch Linux: `pacman -S wl-clipboard`
  - Fedora: `dnf install wl-clipboard`

## Usage

1. Run the `discordo` executable with no arguments.

2. Enter your client authentication token (first-time login) and click on the "Login" button to continue.

- If you are logging in with a bot account, prefix the token with `Bot ` (eg: `discordo --token "Bot OTI2MDU5NTQxNDE2Nzc5ODA2.Yc2KKA.2iZ-5JxgxG-9Ub8GHzBSn-NJjNg"`). Furthermore, it is strongly recommended to change the user agent for HTTP requests from the configuration file if you are using a bot account to log in since the default user agent is that of a browser.

- Most of the Discord third-party clients store the token in a configuration file unencrypted. Discordo securely stores the token in the default OS-specific keyring. 

### Configuration

A default configuration file is created on first start-up at `$HOME/.config/discordo/config.lua` on Unix, `$HOME/Library/Application Support/discordo/config.lua` on Darwin, and `%AppData%/discordo/config.lua` on Windows. You can configure the default configuration directory path using the `config` command-line flag (eg: `discordo --config $HOME/my-custom-dir).

## Disclaimer

Automated user accounts or "self-bots" are against Discord's Terms of Service. I am not responsible for any loss caused by using "self-bots" or Discordo.
