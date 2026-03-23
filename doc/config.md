# Configuration

## auto_focus
- Type: `bool`
- Default: `true`
- Description: Whether to focus the message input automatically when a channel is selected.  
Set to false to preview channels without moving focus.

## mouse
- Type: `bool`
- Default: `true`
- Description: Whether to enable mouse or not.

## editor
- Type: `string`
- Default: `"default"`
- Description: The program to open when the `message_input.editor` keybind is pressed.  
Set it to `"default"` to use `$EDITOR` environment variable.

## status
- Type: `Status`
- Default: `"unknown"`
- Description: Values: "unknown", "online", "dnd", "idle", "invisible", "offline"

## hide_blocked_users
- Type: `bool`
- Default: `true`

## show_attachment_links
- Type: `bool`
- Default: `true`

## autocomplete_limit
- Type: `uint8`
- Default: `20`
- Description: The maximum number of members to populate in the mentions list.  
Set to 0 to disable.

## messages_limit
- Type: `uint8`
- Default: `50`
- Description: The number of messages to fetch when a text-based channel is selected from guilds tree.  
The minimum and maximum value is 1 and 100, respectively.

## markdown

### enabled
- Type: `bool`
- Default: `true`
- Description: Whether to parse and render markdown in messages or not.

### theme
- Type: `string`
- Default: `"monokai"`
- Description: The theme for fenced code blocks.  
Available themes: https://xyproto.github.io/splash/docs.

## help

### compact_modifiers
- Type: `bool`
- Default: `true`
- Description: Show compact key modifiers in help, e.g. "^x" instead of "ctrl+x".

### padding
- Type: `[2]int`
- Default: `[1, 1]`
- Description: The horizontal padding around help content: [left, right].

### separator
- Type: `string`
- Default: `" • "`
- Description: The visual separator between keybinds.

## picker

### width
- Type: `int`
- Default: `80`

### height
- Type: `int`
- Default: `25`

## timestamps

### enabled
- Type: `bool`
- Default: `true`

### format
- Type: `string`
- Default: `"3:04PM"`

## date_separator

### enabled
- Type: `bool`
- Default: `true`

### format
- Type: `string`
- Default: `"January 2, 2006"`

### character
- Type: `string`
- Default: `"─"`

## notifications

### enabled
- Type: `bool`
- Default: `true`

### duration
- Type: `int`
- Default: `0`

### sound

#### enabled
- Type: `bool`
- Default: `true`

#### only_on_ping
- Type: `bool`
- Default: `true`

## typing_indicator

### send
- Type: `bool`
- Default: `true`
- Description: Whether to send typing status or not.

### receive
- Type: `bool`
- Default: `true`
- Description: Whether to receive typing status or not.

## sidebar

### markers

#### expanded
- Type: `string`
- Default: `"▾ "`

#### collapsed
- Type: `string`
- Default: `"▸ "`

#### leaf
- Type: `string`
- Default: `""`

## icons

### guild_category
- Type: `string`
- Default: `""`

### guild_text
- Type: `string`
- Default: `"#"`

### guild_voice
- Type: `string`
- Default: `"♪ "`

### guild_stage_voice
- Type: `string`
- Default: `"♪ "`

### guild_announcement_thread
- Type: `string`
- Default: `"a-"`

### guild_public_thread
- Type: `string`
- Default: `"› "`

### guild_private_thread
- Type: `string`
- Default: `"› "`

### guild_announcement
- Type: `string`
- Default: `"a-"`

### guild_forum
- Type: `string`
- Default: `"≡ "`

### guild_store
- Type: `string`
- Default: `"s-"`

## keybinds
Each keybind field accepts either a string or a list of strings.  
Type: `Keybind = string or []string`.  
```toml  
[keybinds]  
quit = "ctrl+q"  
# or,  
quit = ["ctrl+q", "ctrl+c", "q", ...]  
```

### toggle_guilds_tree
- Type: `Keybind`
- Default: `ctrl+b`

### toggle_channels_picker
- Type: `Keybind`
- Default: `ctrl+k`

### toggle_help
- Type: `Keybind`
- Default: `ctrl+.`

### suspend
- Type: `Keybind`
- Default: `ctrl+z`

### focus_guilds_tree
- Type: `Keybind`
- Default: `ctrl+g`

### focus_messages_list
- Type: `Keybind`
- Default: `ctrl+t`

### focus_message_input
- Type: `Keybind`
- Default: `ctrl+i`

### focus_previous
- Type: `Keybind`
- Default: `ctrl+h`

### focus_next
- Type: `Keybind`
- Default: `ctrl+l`

### picker

#### up
- Type: `Keybind`
- Default: `ctrl+p`

#### down
- Type: `Keybind`
- Default: `ctrl+n`

#### top
- Type: `Keybind`
- Default: `home`

#### bottom
- Type: `Keybind`
- Default: `end`

#### select
- Type: `Keybind`
- Default: `enter`

#### cancel
- Type: `Keybind`
- Default: `esc`

### guilds_tree

#### up
- Type: `Keybind`
- Default: `k`

#### down
- Type: `Keybind`
- Default: `j`

#### top
- Type: `Keybind`
- Default: `g`

#### bottom
- Type: `Keybind`
- Default: `G`

#### select_current
- Type: `Keybind`
- Default: `enter`

#### yank_id
- Type: `Keybind`
- Default: `i`

#### collapse_parent_node
- Type: `Keybind`
- Default: `-`

#### move_to_parent_node
- Type: `Keybind`
- Default: `p`

### messages_list

#### select_up
- Type: `Keybind`
- Default: `k`

#### select_down
- Type: `Keybind`
- Default: `j`

#### select_top
- Type: `Keybind`
- Default: `g`

#### select_bottom
- Type: `Keybind`
- Default: `G`

#### scroll_up
- Type: `Keybind`
- Default: `K`

#### scroll_down
- Type: `Keybind`
- Default: `J`

#### scroll_top
- Type: `Keybind`
- Default: `home`

#### scroll_bottom
- Type: `Keybind`
- Default: `end`

#### select_reply
- Type: `Keybind`
- Default: `s`

#### reply
- Type: `Keybind`
- Default: `R`

#### reply_mention
- Type: `Keybind`
- Default: `r`

#### cancel
- Type: `Keybind`
- Default: `esc`

#### edit
- Type: `Keybind`
- Default: `e`

#### delete
- Type: `Keybind`
- Default: `D`

#### delete_confirm
- Type: `Keybind`
- Default: `d`

#### open
- Type: `Keybind`
- Default: `o`

#### yank_content
- Type: `Keybind`
- Default: `y`

#### yank_url
- Type: `Keybind`
- Default: `u`

#### yank_id
- Type: `Keybind`
- Default: `i`

### message_input

#### paste
- Type: `Keybind`
- Default: `ctrl+v`

#### send
- Type: `Keybind`
- Default: `enter`

#### cancel
- Type: `Keybind`
- Default: `esc`

#### tab_complete
- Type: `Keybind`
- Default: `tab`

#### undo
- Type: `Keybind`
- Default: `ctrl+u`

#### open_editor
- Type: `Keybind`
- Default: `ctrl+e`

#### open_file_picker
- Type: `Keybind`
- Default: `ctrl+\`

### mentions_list

#### up
- Type: `Keybind`
- Default: `ctrl+p`

#### down
- Type: `Keybind`
- Default: `ctrl+n`

#### top
- Type: `Keybind`
- Default: `home`

#### bottom
- Type: `Keybind`
- Default: `end`

### logout
- Type: `Keybind`
- Default: `ctrl+d`

### quit
- Type: `Keybind`
- Default: `ctrl+c`

## theme
Types:  
Alignment = "left" | "center" | "right"  
Attributes = string | []string  
Style = { foreground?, background?, attributes?, underline?, underline_color? }  
BorderSet = "hidden" | "plain" | "round" | "thick" | "double"  
GlyphSet = "minimal" | "box_drawing" | "boxdrawing" | "box" | "unicode"  
ScrollBarVisibility = "automatic" | "auto" | "always" | "never" | "hidden" | "off"

### title

#### normal_style
- Type: `Style`
- Default: `{ attributes = 'dim' }`

#### active_style
- Type: `Style`
- Default: `{ foreground = 'green', attributes = 'bold' }`

#### alignment
- Type: `Alignment`
- Default: `left`

### footer

#### normal_style
- Type: `Style`
- Default: `{ attributes = 'dim' }`

#### active_style
- Type: `Style`
- Default: `{ foreground = 'green', attributes = 'bold' }`

#### alignment
- Type: `Alignment`
- Default: `left`

### border

#### normal_style
- Type: `Style`
- Default: `{ attributes = 'dim' }`

#### active_style
- Type: `Style`
- Default: `{ foreground = 'green', attributes = 'bold' }`

#### enabled
- Type: `bool`
- Default: `true`

#### padding
- Type: `[4]int`
- Default: `[0, 0, 1, 1]`
- Description: Border padding order: [top, right, bottom, left].

#### normal_set
- Type: `BorderSet`
- Default: `round`

#### active_set
- Type: `BorderSet`
- Default: `round`

### guilds_tree

#### auto_expand_folders
- Type: `bool`
- Default: `true`

#### graphics
- Type: `bool`
- Default: `true`

#### graphics_color
- Type: `string`
- Default: `"default"`

#### indents

##### guild
- Type: `int`
- Default: `2`

##### category
- Type: `int`
- Default: `1`

##### channel
- Type: `int`
- Default: `2`

##### forum
- Type: `int`
- Default: `2`

##### group_dm
- Type: `int`
- Default: `1`

##### dm
- Type: `int`
- Default: `2`

### scroll_bar

#### visibility
- Type: `ScrollBarVisibility`
- Default: `auto`

#### glyph_set
- Type: `GlyphSet`
- Default: `unicode`

#### track_style
- Type: `Style`
- Default: `{ attributes = 'dim' }`

#### thumb_style
- Type: `Style`
- Default: `{}`

### messages_list

#### reply_indicator
- Type: `string`
- Default: `">"`

#### forwarded_indicator
- Type: `string`
- Default: `"<"`

#### author_style
- Type: `Style`
- Default: `{}`

#### mention_style
- Type: `Style`
- Default: `{ foreground = 'blue', attributes = 'bold' }`

#### emoji_style
- Type: `Style`
- Default: `{ foreground = 'green' }`

#### url_style
- Type: `Style`
- Default: `{ foreground = 'blue' }`

#### attachment_style
- Type: `Style`
- Default: `{ foreground = 'yellow' }`

#### message_style
- Type: `Style`
- Default: `{}`

#### selected_message_style
- Type: `Style`
- Default: `{ attributes = 'reverse' }`

#### embeds

##### provider_style
- Type: `Style`
- Default: `{ attributes = ['dim', 'italic'] }`

##### author_style
- Type: `Style`
- Default: `{ attributes = 'italic' }`

##### title_style
- Type: `Style`
- Default: `{ foreground = 'blue', attributes = 'bold' }`

##### description_style
- Type: `Style`
- Default: `{ attributes = 'dim' }`

##### field_name_style
- Type: `Style`
- Default: `{ attributes = ['bold', 'underline'] }`

##### field_value_style
- Type: `Style`
- Default: `{}`

##### footer_style
- Type: `Style`
- Default: `{ attributes = ['dim', 'italic'] }`

##### url_style
- Type: `Style`
- Default: `{ foreground = 'blue', underline = 'solid' }`

### mentions_list

#### min_width
- Type: `uint`
- Default: `20`

#### max_height
- Type: `uint`
- Default: `0`

### dialog

#### style
- Type: `Style`
- Default: `{}`

#### background_style
- Type: `Style`
- Default: `{ attributes = 'dim' }`

### help

#### short_key_style
- Type: `Style`
- Default: `{ attributes = 'dim' }`

#### short_desc_style
- Type: `Style`
- Default: `{}`

#### full_key_style
- Type: `Style`
- Default: `{ attributes = 'dim' }`

#### full_desc_style
- Type: `Style`
- Default: `{}`
