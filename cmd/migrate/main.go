package main
import (
"context"
"fmt"
"log"
"os"
"strings"
"github.com/jackc/pgx/v5"
)
func main() {
content, err := os.ReadFile(".env")
if err == nil {
lines := strings.Split(string(content), "\n")
for _, line := range lines {
line = strings.TrimSpace(line)
if line == "" || strings.HasPrefix(line, "#") { continue }
parts := strings.SplitN(line, "=", 2)
if len(parts) == 2 { os.Setenv(parts[0], parts[1]) }
}
}
dbURL := os.Getenv("DATABASE_URL")
if dbURL == "" { log.Fatal("DATABASE_URL is not set") }
ctx := context.Background()
conn, err := pgx.Connect(ctx, dbURL)
if err != nil { log.Fatal(err) }
defer conn.Close(ctx)
sqlBytes, err := os.ReadFile("db/migrations/006_replies_reactions_reads.up.sql")
if err != nil { log.Fatal(err) }
_, err = conn.Exec(ctx, string(sqlBytes))
if err != nil { log.Fatal(err) }
fmt.Println("Migration successful!")
}
