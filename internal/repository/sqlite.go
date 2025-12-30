package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/lehmann314159/vocabulator/internal/models"
)

// SQLiteRepository implements WordRepository using SQLite
type SQLiteRepository struct {
	db *sql.DB
}

// NewSQLiteRepository creates a new SQLite repository
func NewSQLiteRepository(db *sql.DB) *SQLiteRepository {
	return &SQLiteRepository{db: db}
}

// Create inserts a new word and returns the created word with ID
func (r *SQLiteRepository) Create(ctx context.Context, word *models.Word) (*models.Word, error) {
	tagsJSON, err := json.Marshal(word.Tags)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal tags: %w", err)
	}

	now := time.Now()
	result, err := r.db.ExecContext(ctx,
		`INSERT INTO words (word, source, date_learned, part_of_speech, example_sentence, tags, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		word.Word, word.Source, word.DateLearned, word.PartOfSpeech, word.ExampleSentence, string(tagsJSON), now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert word: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert id: %w", err)
	}

	word.ID = id
	word.CreatedAt = now
	word.UpdatedAt = now
	return word, nil
}

// GetByID retrieves a word by its ID
func (r *SQLiteRepository) GetByID(ctx context.Context, id int64) (*models.Word, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, word, source, date_learned, part_of_speech, example_sentence, tags, created_at, updated_at
		 FROM words WHERE id = ?`, id,
	)
	return r.scanWord(row)
}

// GetByWord retrieves a word by the word text itself
func (r *SQLiteRepository) GetByWord(ctx context.Context, word string) (*models.Word, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, word, source, date_learned, part_of_speech, example_sentence, tags, created_at, updated_at
		 FROM words WHERE word = ?`, word,
	)
	return r.scanWord(row)
}

// List retrieves words with optional filtering
func (r *SQLiteRepository) List(ctx context.Context, filter models.WordFilter) ([]*models.Word, error) {
	query, args := r.buildListQuery(filter, false)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query words: %w", err)
	}
	defer rows.Close()

	var words []*models.Word
	for rows.Next() {
		word, err := r.scanWordFromRows(rows)
		if err != nil {
			return nil, err
		}
		words = append(words, word)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return words, nil
}

// Update modifies an existing word
func (r *SQLiteRepository) Update(ctx context.Context, word *models.Word) (*models.Word, error) {
	tagsJSON, err := json.Marshal(word.Tags)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal tags: %w", err)
	}

	now := time.Now()
	_, err = r.db.ExecContext(ctx,
		`UPDATE words SET word = ?, source = ?, date_learned = ?, part_of_speech = ?,
		 example_sentence = ?, tags = ?, updated_at = ? WHERE id = ?`,
		word.Word, word.Source, word.DateLearned, word.PartOfSpeech, word.ExampleSentence,
		string(tagsJSON), now, word.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update word: %w", err)
	}

	word.UpdatedAt = now
	return word, nil
}

// Delete removes a word by ID
func (r *SQLiteRepository) Delete(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM words WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete word: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// GetRandom retrieves a random word
func (r *SQLiteRepository) GetRandom(ctx context.Context) (*models.Word, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, word, source, date_learned, part_of_speech, example_sentence, tags, created_at, updated_at
		 FROM words ORDER BY RANDOM() LIMIT 1`,
	)
	return r.scanWord(row)
}

// Count returns the total number of words matching the filter
func (r *SQLiteRepository) Count(ctx context.Context, filter models.WordFilter) (int64, error) {
	query, args := r.buildListQuery(filter, true)
	var count int64
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count words: %w", err)
	}
	return count, nil
}

// buildListQuery constructs the SQL query for listing words
func (r *SQLiteRepository) buildListQuery(filter models.WordFilter, countOnly bool) (string, []interface{}) {
	var conditions []string
	var args []interface{}

	if filter.Search != "" {
		conditions = append(conditions, "word LIKE ?")
		args = append(args, "%"+filter.Search+"%")
	}

	if filter.Source != "" {
		conditions = append(conditions, "source = ?")
		args = append(args, filter.Source)
	}

	if filter.Tag != "" {
		conditions = append(conditions, "tags LIKE ?")
		args = append(args, "%\""+filter.Tag+"\"%")
	}

	if filter.FromDate != "" {
		conditions = append(conditions, "date_learned >= ?")
		args = append(args, filter.FromDate)
	}

	if filter.ToDate != "" {
		conditions = append(conditions, "date_learned <= ?")
		args = append(args, filter.ToDate)
	}

	var query string
	if countOnly {
		query = "SELECT COUNT(*) FROM words"
	} else {
		query = `SELECT id, word, source, date_learned, part_of_speech, example_sentence, tags, created_at, updated_at FROM words`
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	if !countOnly {
		query += " ORDER BY date_learned DESC, id DESC"

		if filter.Limit > 0 {
			query += fmt.Sprintf(" LIMIT %d", filter.Limit)
		}

		if filter.Offset > 0 {
			query += fmt.Sprintf(" OFFSET %d", filter.Offset)
		}
	}

	return query, args
}

// scanWord scans a single row into a Word struct
func (r *SQLiteRepository) scanWord(row *sql.Row) (*models.Word, error) {
	var word models.Word
	var tagsJSON string
	var partOfSpeech, exampleSentence sql.NullString

	err := row.Scan(
		&word.ID, &word.Word, &word.Source, &word.DateLearned,
		&partOfSpeech, &exampleSentence, &tagsJSON,
		&word.CreatedAt, &word.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, err
		}
		return nil, fmt.Errorf("failed to scan word: %w", err)
	}

	if partOfSpeech.Valid {
		word.PartOfSpeech = &partOfSpeech.String
	}
	if exampleSentence.Valid {
		word.ExampleSentence = &exampleSentence.String
	}

	if err := json.Unmarshal([]byte(tagsJSON), &word.Tags); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tags: %w", err)
	}

	return &word, nil
}

// scanWordFromRows scans a row from sql.Rows into a Word struct
func (r *SQLiteRepository) scanWordFromRows(rows *sql.Rows) (*models.Word, error) {
	var word models.Word
	var tagsJSON string
	var partOfSpeech, exampleSentence sql.NullString

	err := rows.Scan(
		&word.ID, &word.Word, &word.Source, &word.DateLearned,
		&partOfSpeech, &exampleSentence, &tagsJSON,
		&word.CreatedAt, &word.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan word: %w", err)
	}

	if partOfSpeech.Valid {
		word.PartOfSpeech = &partOfSpeech.String
	}
	if exampleSentence.Valid {
		word.ExampleSentence = &exampleSentence.String
	}

	if err := json.Unmarshal([]byte(tagsJSON), &word.Tags); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tags: %w", err)
	}

	return &word, nil
}
