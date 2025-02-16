package storage

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"regexp"
	"time"

	"log"
	"strings"

	"tebexpressapi/pkg/constant"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/spf13/viper"
)

type AmazonS3 struct {
	Session *session.Session
	Domain  string
	Bucket  string
	Region  string
}

type AmazonS3Config struct {
	AccessKeyID     string
	SecretAccessKey string
	Domain          string
	Bucket          string
	Region          string
}

func NewAmazonS3(opts *AmazonS3Config) *AmazonS3 {
	if opts == nil {
		opts = &AmazonS3Config{
			// AccessKeyID:     viper.GetString("aws.ACCESS_KEY_ID"),
			// SecretAccessKey: viper.GetString("aws.SECRET_ACCESS_KEY"),
			// Domain: viper.GetString("aws.S3_DOMAIN"),
			Bucket: viper.GetString("aws.S3_BUCKET"),
			Region: viper.GetString("aws.S3_REGION"),
		}
	}

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(opts.Region),
		// Credentials: credentials.NewStaticCredentials(opts.AccessKeyID, opts.SecretAccessKey, ""),
	})

	if err != nil {
		log.Printf("Get session AWS errors, %v", err)
		return nil
	}

	return &AmazonS3{
		Session: sess,
		Domain:  opts.Domain,
		Bucket:  opts.Bucket,
		Region:  opts.Region,
	}
}

func (s *AmazonS3) UploadFile(file io.Reader, filename string, bucket string, contentType string) error {
	bf, err := ioutil.ReadAll(file)
	if err != nil {
		return err
	}

	if contentType == "" {
		contentType = http.DetectContentType(bf)
	}

	_, err = s3.New(s.Session).PutObject(&s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(filename),
		//ACL:                  aws.String("public-read"),
		Body:                 bytes.NewReader(bf),
		ContentLength:        aws.Int64(int64(len(bf))),
		ContentType:          aws.String(contentType),
		ContentDisposition:   aws.String("attachment"),
		ServerSideEncryption: aws.String("AES256"),
		ContentEncoding:      aws.String("utf-8"),
	})

	return err
}

func (s *AmazonS3) ReadFile(url string, bucket string) (*bytes.Buffer, error) {
	key := s.GetKeyFromUrl(url)
	results, err := s3.New(s.Session).GetObject(&s3.GetObjectInput{
		Bucket: aws.String(s.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return bytes.NewBuffer(nil), err
	}
	defer results.Body.Close()

	buf := bytes.NewBuffer(nil)
	if _, err := io.Copy(buf, results.Body); err != nil {
		return bytes.NewBuffer(nil), err
	}

	return buf, nil
}

func (s *AmazonS3) Read(url string, bucket string) (*s3.GetObjectOutput, error) {
	key := s.GetKeyFromUrl(url)
	results, err := s3.New(s.Session).GetObject(&s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})

	return results, err
}

func (s *AmazonS3) UploadChunkToS3(fileName, path, bucket, contentType string, header []string, data [][]string) error {
	return nil
}

func (s *AmazonS3) Exist(url string, bucket string) (bool, error) {
	key := s.GetKeyFromUrl(url)
	_, err := s3.New(s.Session).HeadObject(&s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return false, err
	}

	return true, nil
}

func (s *AmazonS3) Delete(listKey []string, bucket string) error {
	objectsToDelete := make([]*s3.ObjectIdentifier, 0, 1000)

	for _, url := range listKey {
		key := s.GetKeyFromUrl(url)
		obj := s3.ObjectIdentifier{
			Key: aws.String(key),
		}
		objectsToDelete = append(objectsToDelete, &obj)
	}
	deleteArray := s3.Delete{Objects: objectsToDelete}
	input := &s3.DeleteObjectsInput{
		Bucket: aws.String(bucket),
		Delete: &deleteArray,
	}

	_, err := s3.New(s.Session).DeleteObjects(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				log.Printf("Delete AWS errors, %v", err)
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			log.Printf("Delete AWS errors, %v", err)
		}
		return err
	}

	return nil
}

func (s *AmazonS3) GetKeyFromUrl(url string) string {
	key := url
	reg := regexp.MustCompile("^(" + s.Domain + ")")
	if s.Domain != "" && reg.MatchString(url) {
		key = reg.ReplaceAllString(key, "")
	} else {
		reg = regexp.MustCompile(`(?mi)^http(s)?:\/\/([a-zA-Z0-9-_.]+)\.amazonaws\.com\/([a-zA-Z0-9-_\.]+)`)
		key = reg.ReplaceAllString(key, "")
	}

	return key
}

func (s *AmazonS3) GetThumbnailKey(url string) string {
	key := s.GetKeyFromUrl(url)
	return strings.Replace(constant.PathThumbnail, "/", "", 1) + key
}

func (s *AmazonS3) ReadFileForUpload(filename string, bucket string) (*bytes.Buffer, error, *string) {
	results, err := s3.New(s.Session).GetObject(&s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(filename),
	})

	contentType := results.ContentType

	if err != nil {
		return bytes.NewBuffer(nil), err, nil
	}

	defer results.Body.Close()
	buf := bytes.NewBuffer(nil)

	if _, err := io.Copy(buf, results.Body); err != nil {
		return bytes.NewBuffer(nil), err, nil
	}

	return buf, nil, contentType
}

func (s *AmazonS3) PreAssign(filename string, bucket string) (string, error) {
	s3Client := s3.New(s.Session)
	data, _ := s3Client.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(filename),
	})

	urlStr, err := data.Presign(15 * time.Minute)
	if err != nil {
		return "", err
	}

	return urlStr, nil
}
