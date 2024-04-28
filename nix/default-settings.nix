{
  mouse = true;
  timestamps = false;
  timestamps_before_author = false;
  timestamps_format = "3:04PM";

  messages_limit = 50;
  editor = "default";

  keys = {
    focus_guilds_tree = "Ctrl+G";
    focus_messages_text = "Ctrl+T";
    focus_message_input = "Ctrl+P";
    toggle_guild_tree = "Ctrl+B";
    select_previous = "Rune[k]";
    select_next = "Rune[j]";
    select_first = "Rune[g]";
    select_last = "Rune[G]";

    guilds_tree = {
      select_current = "Enter";
    };

    messages_text = {
      select_reply = "Rune[s]";
      reply = "Rune[r]";
      reply_mention = "Rune[R]";
      delete = "Rune[d]";
      yank = "Rune[y]";
      open = "Rune[o]";
    };

    message_input = {
      send = "Enter";
      editor = "Ctrl+E";
      cancel = "Esc";
    };
  };

  theme = {
    border = true;
    border_color = "default";
    border_padding = [ 0 0 1 1 ];
    title_color = "default";
    background_color = "default";

    guilds_tree = {
      auto_expand_folders = true;
      graphics = true;
    };

    messages_text = {
      author_color = "aqua";
      reply_indicator = "╭ ";
    };
  };
}





