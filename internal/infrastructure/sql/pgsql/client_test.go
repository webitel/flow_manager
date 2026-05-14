//go:build integration

package pgsql

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/webitel/wlog"

	"github.com/webitel/flow_manager/internal/infrastructure/sql"
)

var testStore sql.Store

// testUser - це структура для тестування.
type testUser struct {
	ID   int    `db:"id"`
	Name string `db:"name"`
	Age  int    `db:"age"`
}

// TestMain - це головна функція для налаштування тестового середовища.
func TestMain(m *testing.M) {
	// Потрібно, щоб Docker був запущений
	ctx := context.Background()

	// 1. Створюємо контейнер PostgreSQL
	pgContainer, err := postgres.RunContainer(ctx,
		testcontainers.WithImage("docker.io/pgvector/pgvector:pg16"),
		postgres.WithDatabase("test-db"),
		postgres.WithUsername("user"),
		postgres.WithPassword("password"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(5*time.Minute),
		),
	)
	if err != nil {
		fmt.Printf("Could not start postgres container: %s", err)
		os.Exit(1)
	}

	_, _, err = pgContainer.Exec(ctx, []string{"psql", "test-db", "-U", "user", "-c", "CREATE EXTENSION IF NOT EXISTS vector;"})
	if err != nil {
		fmt.Printf("Could not set vector: %s", err)
		os.Exit(1)
	}

	// 2. Отримуємо DSN (рядок підключення) до нашої тестової бази
	dsn, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		fmt.Printf("Could not get connection string: %s", err)
		os.Exit(1)
	}

	// 3. Ініціалізуємо наше сховище
	testStore, err = New(ctx, dsn, wlog.NewLogger(&wlog.LoggerConfiguration{
		EnableConsole: true,
	}))
	if err != nil {
		fmt.Printf("Could not connect to test database: %s", err)
		os.Exit(1)
	}

	// 4. Створюємо таблицю для тестів
	_, err = testStore.Query(ctx, `
        CREATE TABLE test_users (
            id SERIAL PRIMARY KEY,
            name TEXT NOT NULL,
            age INT NOT NULL
        );
    `, nil)
	if err != nil {
		fmt.Printf("Could not create test table: %s", err)
		os.Exit(1)
	}
	// 5. Запускаємо тести
	code := m.Run()

	// 6. Зупиняємо та видаляємо контейнер після завершення тестів
	if err := pgContainer.Terminate(ctx); err != nil {
		fmt.Printf("Could not terminate postgres container: %s", err)
	}

	os.Exit(code)
}

func TestDB_Exec_And_Get(t *testing.T) {
	ctx := context.Background()

	// Очищуємо таблицю перед тестом для ізоляції
	_, err := testStore.Query(ctx, "TRUNCATE TABLE test_users RESTART IDENTITY", nil)
	require.NoError(t, err)

	t.Run("Get existing user", func(t *testing.T) {
		// --- Arrange ---
		userName := "Ihor"
		userAge := 30
		insertQuery := "INSERT INTO test_users (name, age) VALUES (@name, @age)"
		insertArgs := pgx.NamedArgs{"name": userName, "age": userAge}

		// --- Act ---
		// Використовуємо Exec для вставки даних
		err := testStore.Exec(ctx, insertQuery, insertArgs)
		require.NoError(t, err, "Exec should not return an error")

		// Використовуємо Get для отримання вставленого запису
		var retrievedUser testUser

		getQuery := "SELECT * FROM test_users WHERE name = @name"
		getArgs := pgx.NamedArgs{"name": userName}
		err = testStore.Get(ctx, &retrievedUser, getQuery, getArgs)

		// --- Assert ---
		require.NoError(t, err, "Get should not return an error for existing user")
		assert.Equal(t, 1, retrievedUser.ID) // ID має бути 1, бо ми зробили RESTART IDENTITY
		assert.Equal(t, userName, retrievedUser.Name)
		assert.Equal(t, userAge, retrievedUser.Age)
	})

	t.Run("Get non-existing user", func(t *testing.T) {
		// --- Arrange ---
		var retrievedUser testUser

		getQuery := "SELECT * FROM test_users WHERE name = @name"
		getArgs := pgx.NamedArgs{"name": "NonExistentUser"}

		// --- Act ---
		err := testStore.Get(ctx, &retrievedUser, getQuery, getArgs)

		// --- Assert ---
		// pgxscan.Get повертає pgx.ErrNoRows, коли запис не знайдено
		require.Error(t, err, "Get should return an error for non-existing user")
		assert.ErrorIs(t, err, pgx.ErrNoRows, "Error should be pgx.ErrNoRows")
	})
}

func TestDB_Select(t *testing.T) {
	ctx := context.Background()

	// --- Arrange ---
	// Очищуємо таблицю і вставляємо тестові дані
	_, err := testStore.Query(ctx, "TRUNCATE TABLE test_users RESTART IDENTITY", nil)
	require.NoError(t, err)

	usersToInsert := []testUser{
		{Name: "Alice", Age: 25},
		{Name: "Bob", Age: 35},
		{Name: "Charlie", Age: 45},
	}

	for _, u := range usersToInsert {
		err := testStore.Exec(ctx, "INSERT INTO test_users (name, age) VALUES (@name, @age)", pgx.NamedArgs{
			"name": u.Name,
			"age":  u.Age,
		})
		require.NoError(t, err)
	}

	// --- Act ---
	var retrievedUsers []testUser

	err = testStore.Select(ctx, &retrievedUsers, "SELECT * FROM test_users ORDER BY name", nil)

	// --- Assert ---
	require.NoError(t, err, "Select should not return an error")
	require.Len(t, retrievedUsers, 3, "Select should return 3 users")

	assert.Equal(t, "Alice", retrievedUsers[0].Name)
	assert.Equal(t, "Bob", retrievedUsers[1].Name)
	assert.Equal(t, "Charlie", retrievedUsers[2].Name)
}

func TestDB_Query(t *testing.T) {
	ctx := context.Background()

	// --- Arrange ---
	_, err := testStore.Query(ctx, "TRUNCATE TABLE test_users RESTART IDENTITY", nil)
	require.NoError(t, err)
	err = testStore.Exec(ctx, "INSERT INTO test_users (name, age) VALUES ('David', 50)", nil)
	require.NoError(t, err)

	// --- Act ---
	rows, err := testStore.Query(ctx, "SELECT id, name, age FROM test_users WHERE name = 'David'", nil)
	require.NoError(t, err, "Query should not return an error")

	defer rows.Close()

	// --- Assert ---
	require.True(t, rows.Next(), "Should be one row in the result set")

	var user testUser

	err = rows.Scan(&user.ID, &user.Name, &user.Age)
	require.NoError(t, err, "Scan should not return an error")

	assert.Equal(t, "David", user.Name)
	assert.Equal(t, 50, user.Age)

	// Переконуємось, що більше рядків немає
	assert.False(t, rows.Next(), "Should be no more rows")
	// Перевіряємо, чи не було помилок під час ітерації
	require.NoError(t, rows.Err())
}
