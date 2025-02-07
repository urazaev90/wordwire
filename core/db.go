// взаимодействие с базой данных

package core

import (
	"context"
	"github.com/gorilla/sessions"
	"github.com/jackc/pgx/v5/pgxpool"
	"log"
	"time"
)

var (
	Database *pgxpool.Pool
	store    = sessions.NewCookieStore([]byte("jdfH=S5Ds+SFg4ff)-dfdWg2gD7D+Ddhdf"))
)

// регистрация даты последнего посещения пользователя
func updateLastVisitDate(userID int) {
	ctx := context.Background() // Или используйте нужный контекст
	_, err := Database.Exec(ctx, `
		UPDATE user_accounts 
		SET last_visit_date = $1 
		WHERE id = $2`,
		time.Now().Format("2006-01-02"), userID)
	if err != nil {
		log.Println("Error updating last visit date:", err)
	}
}

// запрос для добавления слова в архив (надо уточнить)
func loadNextWordForUser(userID int) error {
	// Выполняем запрос для добавления следующего слова
	ctx := context.Background() // Или используйте нужный контекст
	_, err := Database.Exec(ctx, `
        INSERT INTO user_word_labels (user_id, word_id, label)
        SELECT $1, id, 1
        FROM english_words
        WHERE id NOT IN (SELECT word_id FROM user_word_labels WHERE user_id = $1)
        ORDER BY usage_per_billion DESC
        LIMIT 1
    `, userID)

	return err
}

// запрос на 10 слов для следующей страницы словаря (надо уточнить)
func loadNextPageWordsForUser(userID int) error {
	// Выполняем запрос для добавления 10 следующих слов
	ctx := context.Background()
	_, err := Database.Exec(ctx, `
        INSERT INTO user_word_labels (user_id, word_id, label)
        SELECT $1, id, 1
        FROM english_words
        WHERE id NOT IN (SELECT word_id FROM user_word_labels WHERE user_id = $1)
        ORDER BY usage_per_billion DESC
        LIMIT 10
    `, userID)

	return err
}

// возвращаем количество выбранных и архивированных слов
func getWordCountsAsync(userID int, ch chan<- map[string]int) {
	counts := make(map[string]int)

	var selectedCount, archivedCount int

	ctx := context.Background()
	err := Database.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM user_word_labels
		WHERE user_id = $1 AND label = 2
	`, userID).Scan(&selectedCount)
	if err == nil {
		counts["selected"] = selectedCount
	}

	err = Database.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM user_word_labels
		WHERE user_id = $1 AND label = 3
	`, userID).Scan(&archivedCount)
	if err == nil {
		counts["archived"] = archivedCount
	}

	ch <- counts
}
