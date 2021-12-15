# Words Database

This directory contains data derived from Wordnet:
Princeton University "About WordNet." [WordNet](https://wordnet.princeton.edu/). Princeton University. 2010.

WordNet 3.0 Copyright 2006 by Princeton University. All rights reserved.
THIS SOFTWARE AND DATABASE IS PROVIDED "AS IS" AND PRINCETON UNIVERSITY MAKES NO REPRESENTATIONS OR WARRANTIES, EXPRESS OR IMPLIED. BY WAY OF EXAMPLE, BUT NOT LIMITATION, PRINCETON UNIVERSITY MAKES NO REPRESENTATIONS OR WARRANTIES OF MERCHANT- ABILITY OR FITNESS FOR ANY PARTICULAR PURPOSE OR THAT THE USE OF THE LICENSED SOFTWARE, DATABASE OR DOCUMENTATION WILL NOT INFRINGE ANY THIRD PARTY PATENTS, COPYRIGHTS, TRADEMARKS OR OTHER RIGHTS. The name of Princeton University or Princeton may not be used in advertising or publicity pertaining to distribution of the software and/or database. Title to copyright in this software, database and any associated documentation shall at all times remain with Princeton University and LICENSEE agrees to preserve same.

# Production

The file words.json.gz was created by loading the Wordnet synsets via Python [NLTK](https://www.nltk.org/).

A JSON file was produced in the following format:

```json
{
    "<POS>": ["array", "of", "words"]
}
```

Where `<POS>` is a part of speech signifier in singular form such as "noun" and "adjective".

# Contents

The current words.json.gz contains the following word counts:

| POS | Count |
| --- | ----- |
| noun/noun phrase | 81,978 |
| adjective | 18,133 |

Generated environment names are of the form `<adjective>-<noun/noun phrase>`.

This allows 1,490,879,940 possible unique environment names.

# Replacement

If you want to replace the bundled word set with your own, replace words.json.gz with your own gzip-compressed JSON file with "noun" and "adjective" word arrays. Ensure that you have enough words to allow
sufficient unique names for your installation, or environment creation will fail. Note that any glyphs that are not DNS-legal (do not match `[A-Za-z0-9\-_]`) will be filtered out.
