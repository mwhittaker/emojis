# Emojis

This directory contains a list of all emojis and emoji sequences, along with
their names, groups, subgroups, and tags. The data is parsed from
[emoji-test.txt][emoji-test] from unicode.org and augmented with tags from
[data.json][data-json] from emojibase.dev.

There are many other emoji datasets:

- https://github.com/muan/emojilib/blob/main/dist/emoji-en-US.json
- https://github.com/github/gemoji/blob/master/db/emoji.json
- https://unpkg.com/emoji.json@14.0.0/emoji.json

This dataset differs in that it only includes fully qualified emoji sequences
but otherwise includes all sequences. Other datasets tend to omit skin tone
variations or include unqualified emojis.

[emoji-test]: https://unicode.org/Public/emoji/latest/emoji-test.txt
[data-json]: https://cdn.jsdelivr.net/npm/emojibase-data@7.0.1/en/data.json
