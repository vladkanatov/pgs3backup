package dump

import (
	"archive/tar"
	"bytes"
	"context"
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver
)

// Dumper реализует создание логического бэкапа без вызова внешних утилит.
// Он формирует tar.gz поток, содержащий файл schema.sql и папку data/ с CSV файлами по таблицам.
type Dumper struct {
	host     string
	port     int
	dbName   string
	user     string
	password string
	sslMode  string
}

// New создает новый Dumper
func New(host string, port int, dbName, user, password string) *Dumper {
	return &Dumper{
		host:     host,
		port:     port,
		dbName:   dbName,
		user:     user,
		password: password,
		sslMode:  "disable",
	}
}

// Dump формирует tar.gz поток со схемой и данными (CSV).
// Возвращает io.ReadCloser, который нужно закрыть после использования.
func (d *Dumper) Dump() (io.ReadCloser, error) {
	pr, pw := io.Pipe()

	go func() {
		defer func() {
			if closeErr := pw.Close(); closeErr != nil {
				pw.CloseWithError(closeErr)
			}
		}()

		// Создаём tar writer
		tw := tar.NewWriter(pw)
		defer func() {
			if closeErr := tw.Close(); closeErr != nil {
				pw.CloseWithError(closeErr)
			}
		}()

		// Открываем DB соединение
		connStr := d.buildConnString()
		db, err := sql.Open("postgres", connStr)
		if err != nil {
			pw.CloseWithError(fmt.Errorf("ошибка открытия подключения к БД: %w", err))
			return
		}
		defer func() {
			if closeErr := db.Close(); closeErr != nil {
				pw.CloseWithError(closeErr)
			}
		}()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		// 1) Собираем список таблиц
		tables, err := d.listTables(ctx, db)
		if err != nil {
			pw.CloseWithError(fmt.Errorf("ошибка получения списка таблиц: %w", err))
			return
		}

		// 2) Формируем schema.sql в памяти
		var schemaBuf bytes.Buffer
		for _, t := range tables {
			create, err := d.buildCreateTable(ctx, db, t.schema, t.name)
			if err != nil {
				pw.CloseWithError(fmt.Errorf("ошибка построения CREATE TABLE для %s.%s: %w", t.schema, t.name, err))
				return
			}
			schemaBuf.WriteString(create)
			schemaBuf.WriteString("\n\n")
		}

		// Пишем schema.sql в tar
		if err := writeTarFile(tw, "schema.sql", schemaBuf.Bytes()); err != nil {
			pw.CloseWithError(fmt.Errorf("ошибка записи schema.sql в tar: %w", err))
			return
		}

		// 3) Для каждой таблицы пишем CSV
		for _, t := range tables {
			filename := fmt.Sprintf("data/%s.%s.csv", t.schema, t.name)
			if err := d.writeTableCSV(ctx, db, tw, t.schema, t.name, filename); err != nil {
				pw.CloseWithError(fmt.Errorf("ошибка экспорта таблицы %s.%s: %w", t.schema, t.name, err))
				return
			}
		}
	}()

	return pr, nil
}

type tableRef struct {
	schema string
	name   string
}

func (d *Dumper) buildConnString() string {
	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(d.user, d.password),
		Host:   fmt.Sprintf("%s:%d", d.host, d.port),
		Path:   d.dbName,
	}
	q := u.Query()
	q.Set("sslmode", d.sslMode)
	u.RawQuery = q.Encode()
	return u.String()
}

func (d *Dumper) listTables(ctx context.Context, db *sql.DB) ([]tableRef, error) {
	rows, err := db.QueryContext(ctx, "SELECT table_schema, table_name FROM information_schema.tables WHERE table_type='BASE TABLE' AND table_schema NOT IN ('pg_catalog','information_schema') ORDER BY table_schema, table_name")
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	var res []tableRef
	for rows.Next() {
		var s, n string
		if err := rows.Scan(&s, &n); err != nil {
			return nil, err
		}
		res = append(res, tableRef{schema: s, name: n})
	}
	return res, rows.Err()
}

func (d *Dumper) buildCreateTable(ctx context.Context, db *sql.DB, schema, table string) (string, error) {
	q := `SELECT column_name, data_type, is_nullable, column_default FROM information_schema.columns WHERE table_schema=$1 AND table_name=$2 ORDER BY ordinal_position`
	rows, err := db.QueryContext(ctx, q, schema, table)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = rows.Close()
	}()

	var cols []string
	for rows.Next() {
		var name, dtype, isNull sql.NullString
		var def sql.NullString
		if err := rows.Scan(&name, &dtype, &isNull, &def); err != nil {
			return "", err
		}
		part := fmt.Sprintf("\t\"%s\" %s", name.String, dtype.String)
		if def.Valid && def.String != "" {
			part += fmt.Sprintf(" DEFAULT %s", def.String)
		}
		if isNull.Valid && isNull.String == "NO" {
			part += " NOT NULL"
		}
		cols = append(cols, part)
	}

	create := fmt.Sprintf("CREATE TABLE \"%s\".\"%s\" (\n%s\n);", schema, table, strings.Join(cols, ",\n"))
	return create, nil
}

func (d *Dumper) writeTableCSV(ctx context.Context, db *sql.DB, tw *tar.Writer, schema, table, filename string) error {
	fq := fmt.Sprintf("\"%s\".\"%s\"", schema, table)
	q := fmt.Sprintf("SELECT * FROM %s", fq)
	rows, err := db.QueryContext(ctx, q)
	if err != nil {
		return err
	}
	defer func() {
		_ = rows.Close()
	}()

	cols, err := rows.Columns()
	if err != nil {
		return err
	}

	// Подготовка tar entry
	// Поскольку размер заранее неизвестен, не указываем Size (0), но tar хочет Size; вместо этого мы буферизуем запись страницы.
	// Для простоты запишем в буфер и затем в tar (может потребовать много памяти для больших таблиц).
	var buf bytes.Buffer
	cw := csv.NewWriter(&buf)
	if err := cw.Write(cols); err != nil {
		return err
	}

	vals := make([]interface{}, len(cols))
	ptrs := make([]interface{}, len(cols))
	for i := range vals {
		ptrs[i] = &vals[i]
	}

	for rows.Next() {
		if err := rows.Scan(ptrs...); err != nil {
			return err
		}
		rec := make([]string, len(cols))
		for i, v := range vals {
			if v == nil {
				rec[i] = ""
				continue
			}
			switch t := v.(type) {
			case []byte:
				rec[i] = string(t)
			default:
				rec[i] = fmt.Sprint(t)
			}
		}
		if err := cw.Write(rec); err != nil {
			return err
		}
	}
	cw.Flush()
	if err := cw.Error(); err != nil {
		return err
	}

	hdr := &tar.Header{
		Name:    filename,
		Mode:    0600,
		Size:    int64(buf.Len()),
		ModTime: time.Now(),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}
	if _, err := tw.Write(buf.Bytes()); err != nil {
		return err
	}
	return rows.Err()
}

func writeTarFile(tw *tar.Writer, name string, data []byte) error {
	hdr := &tar.Header{
		Name:    name,
		Mode:    0600,
		Size:    int64(len(data)),
		ModTime: time.Now(),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}
	_, err := tw.Write(data)
	return err
}
