package storage

import (
	"bytes"
	"io"

	"github.com/aws/aws-sdk-go/service/s3"
)

const (
	AwsType   = "aws"
	LocalType = "local"
)

// S3 --
type S3 interface {
	UploadFile(file io.Reader, filename string, bucket string, contentType string) error
	ReadFile(filename string, bucket string) (*bytes.Buffer, error)
	ReadFileForUpload(filename string, bucket string) (*bytes.Buffer, error, *string)
	PreAssign(filename string, bucket string) (string, error)
	Exist(url string, bucket string) (bool, error)
	Delete(listKey []string, bucket string) error
	Read(url string, bucket string) (*s3.GetObjectOutput, error)
}

type S3Configuration struct {
	Type string
}
