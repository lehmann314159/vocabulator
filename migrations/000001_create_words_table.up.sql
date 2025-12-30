CREATE TABLE IF NOT EXISTS words (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    word TEXT NOT NULL UNIQUE,
    source TEXT NOT NULL,
    date_learned TEXT NOT NULL,
    part_of_speech TEXT,
    example_sentence TEXT,
    tags TEXT DEFAULT '[]',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_words_date_learned ON words(date_learned);
CREATE INDEX IF NOT EXISTS idx_words_source ON words(source);
CREATE INDEX IF NOT EXISTS idx_words_word ON words(word);
