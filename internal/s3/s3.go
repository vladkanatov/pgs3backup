// Package s3 provides S3-compatible storage upload functionality.
package s3

import (
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

// Uploader handles file uploads to S3-compatible storage.
type Uploader struct {
	bucket   string
	uploader *s3manager.Uploader
}

// New создает новый S3 uploader
func New(bucket, region, accessKey, secretKey, endpoint string) (*Uploader, error) {
	config := &aws.Config{
		Region:      aws.String(region),
		Credentials: credentials.NewStaticCredentials(accessKey, secretKey, ""),
	}

	// Если указан custom endpoint (например, MinIO)
	if endpoint != "" {
		config.Endpoint = aws.String(endpoint)
		config.S3ForcePathStyle = aws.Bool(true)
	}

	sess, err := session.NewSession(config)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания AWS сессии: %w", err)
	}

	return &Uploader{
		bucket:   bucket,
		uploader: s3manager.NewUploader(sess),
	}, nil
}

// Upload загружает данные в S3
func (u *Uploader) Upload(reader io.Reader, prefix, dbName string, compressed bool) (string, error) {
	// Генерируем имя файла с timestamp
	timestamp := time.Now().Format("2006-01-02_15-04-05")

	var key string
	if compressed {
		key = fmt.Sprintf("%s/%s_%s.dump.gz", prefix, dbName, timestamp)
	} else {
		key = fmt.Sprintf("%s/%s_%s.dump", prefix, dbName, timestamp)
	}

	// Загружаем в S3
	result, err := u.uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(u.bucket),
		Key:    aws.String(key),
		Body:   reader,
	})

	if err != nil {
		return "", fmt.Errorf("ошибка загрузки в S3: %w", err)
	}

	return result.Location, nil
}

// UploadWithKey загружает данные в S3 с конкретным ключом
func (u *Uploader) UploadWithKey(reader io.Reader, key string) (string, error) {
	result, err := u.uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(u.bucket),
		Key:    aws.String(key),
		Body:   reader,
	})

	if err != nil {
		return "", fmt.Errorf("ошибка загрузки в S3: %w", err)
	}

	return result.Location, nil
}
