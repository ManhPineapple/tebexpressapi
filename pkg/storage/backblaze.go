package storage

import (
	"bytes"
	"context"
	"io"

	"log"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/spf13/viper"
)

type Backblaze struct {
	Client *minio.Client
}

type BackblazeConfig struct {
	AccessKeyID     string
	SecretAccessKey string
	Domain          string
	Bucket          string
	Region          string
}

func NewBackblaze(opts *BackblazeConfig) *Backblaze {
	if opts == nil {
		opts = &BackblazeConfig{
			AccessKeyID:     viper.GetString("b2.ACCESS_KEY_ID"),
			SecretAccessKey: viper.GetString("b2.SECRET_ACCESS_KEY"),
			Domain:          viper.GetString("b2.DOMAIN"),
		}
	}

	useSSL := true

	// Initialize minio client object.
	minioClient, err := minio.New(opts.Domain, &minio.Options{
		Creds:  credentials.NewStaticV4(opts.AccessKeyID, opts.SecretAccessKey, ""),
		Secure: useSSL,
	})

	if err != nil {
		log.Printf("Get session AWS errors, %v", err)
		return nil
	}

	return &Backblaze{
		Client: minioClient,
	}
}

func (s *Backblaze) UploadFile(file io.Reader, filename string, size int64, bucket string, contentType string) error {
	// Upload the file to Backblaze B2 bucket
	_, err := s.Client.PutObject(context.Background(), bucket, filename, file, size, minio.PutObjectOptions{})
	if err != nil {
		log.Fatalln(err)
	}

	return err
}

func (s *Backblaze) ReadFile(url string, bucket string) (*bytes.Buffer, error) {
	object, err := s.Client.GetObject(context.Background(), bucket, url, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	defer object.Close()

	buf := bytes.NewBuffer(nil)

	if _, err := io.Copy(buf, object); err != nil {
		return nil, err
	}

	return buf, nil
}

func (s *Backblaze) Read(url string, bucket string) (*minio.Object, *minio.ObjectInfo, error) {
	object, err := s.Client.GetObject(context.Background(), bucket, url, minio.GetObjectOptions{})
	if err != nil {
		return nil, nil, err
	}

	defer object.Close()
	stat, err := object.Stat()
	if err != nil {
		return nil, nil, err
	}

	return object, &stat, nil
}

func (s *Backblaze) UploadChunkToS3(fileName, path, bucket, contentType string, header []string, data [][]string) error {
	return nil
}

func (s *Backblaze) Exist(url string, bucket string) (bool, error) {
	return true, nil
}

func (s *Backblaze) Delete(listKey []string, bucket string) error {
	return nil
}

func (s *Backblaze) ReadFileForUpload(filename string, bucket string) (*bytes.Buffer, error, *string) {
	object, err := s.Client.GetObject(context.Background(), bucket, filename, minio.GetObjectOptions{})
	if err != nil {
		return nil, err, nil
	}

	defer object.Close()
	stat, err := object.Stat()
	if err != nil {
		return nil, err, nil
	}

	contentType := stat.ContentType
	buf := bytes.NewBuffer(nil)

	if _, err := io.Copy(buf, object); err != nil {
		return nil, err, nil
	}

	return buf, nil, &contentType
}

func (s *Backblaze) PreAssign(filename string, bucket string) (string, error) {
	return "", nil
}
