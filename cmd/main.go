// Package main provides the entry point for pgs3backup utility.
package main

import (
	"fmt"
	"log"

	"github.com/vladkanatov/pgs3backup/internal/compress"
	"github.com/vladkanatov/pgs3backup/internal/config"
	"github.com/vladkanatov/pgs3backup/internal/dump"
	"github.com/vladkanatov/pgs3backup/internal/s3"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("Ошибка: %v", err)
	}
}

func run() error {
	// Загружаем конфигурацию
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("ошибка загрузки конфигурации: %w", err)
	}

	log.Printf("Начинаем бэкап базы данных: %s", cfg.DBName)

	// Создаем dumper
	dumper := dump.New(
		cfg.DBHost,
		cfg.DBPort,
		cfg.DBName,
		cfg.DBUser,
		cfg.DBPassword,
	)

	// Создаем дамп
	log.Println("Создаем дамп базы данных...")
	dumpReader, err := dumper.Dump()
	if err != nil {
		return fmt.Errorf("ошибка создания дампа: %w", err)
	}
	defer func() {
		if closeErr := dumpReader.Close(); closeErr != nil {
			log.Printf("Ошибка закрытия dumpReader: %v", closeErr)
		}
	}()

	// Готовим reader для загрузки
	uploadReader := dumpReader

	// Сжимаем если нужно
	if cfg.Compress {
		log.Println("Сжимаем дамп...")
		compressedReader, err := compress.NewCompressedReader(dumpReader)
		if err != nil {
			return fmt.Errorf("ошибка сжатия: %w", err)
		}
		uploadReader = compressedReader
	}

	// Создаем S3 uploader
	uploader, err := s3.New(
		cfg.S3Bucket,
		cfg.S3Region,
		cfg.S3AccessKey,
		cfg.S3SecretKey,
		cfg.S3Endpoint,
	)
	if err != nil {
		return fmt.Errorf("ошибка создания S3 uploader: %w", err)
	}

	// Загружаем в S3
	log.Println("Загружаем в S3...")
	location, err := uploader.Upload(
		uploadReader,
		cfg.BackupPrefix,
		cfg.DBName,
		cfg.Compress,
	)
	if err != nil {
		return fmt.Errorf("ошибка загрузки в S3: %w", err)
	}

	log.Printf("✓ Бэкап успешно создан: %s", location)
	return nil
}
