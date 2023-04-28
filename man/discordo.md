discordo 1
=======================================

NAME
----

`discordo` - A lightweight, secure, and feature-rich Discord terminal client

SYNOPSIS
--------

`discordo` [`--token "TOKEN"`] [`--config "OPTIONAL/PATH/HERE.yml"`]

DESCRIPTION
-----------

`discordo` is a lightweight, secure, and feature-rich Discord terminal client.  

If no token is provided, you will be prompted with an interactive login.  
The token is stored securely in the default OS-specific keyring.

USAGE
-------

The default keybindings in the app are as follow  

### Guilds Tree

| Action | Keybinding |
| ------ | ---------- |
| Focus  | Alt + g    |

### Messages Text

| Action                | Keybinding |
| --------------------- | ---------- |
| Focus                 | Alt + m    |
| Show image            | i          |
| Copy message content  | c          |
| Reply without mention | r          |
| Reply with mention    | R          |
| Select reply          | s          |
| Reply previous        | Up arrow   |
| Select next           | Down arrow |
| Select first          | Home       |
| Select last           | End        |

### Message Input

| Action               | Keybinding |
| -------------------- | ---------- |
| Focus                | Alt + i    |
| Send message         | Enter      |
| Paste from clipboard | Ctrl + v   |
| Launch editor        | Ctrl + e   |

FILES
-----

*$HOME/.config/discordo/config.yml*
  The configuration file.

*$HOME/.cache/discordo/logs.txt*
  The log file.
