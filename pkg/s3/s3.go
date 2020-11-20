package s3

import (
	"io"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

// Options contains the information needed to push/pull an image from S3
type Options struct {
	Region, Bucket, Key                        string
	Concurrency, PresignTTLMinutes, MaxRetries uint
}

// StorageManager is an object capable of pushing/pulling from S3
type StorageManager struct {
	// LogFunc is function that logs a formatted string somewhere
	LogFunc func(string, ...interface{})

	creds *credentials.Credentials
}

// SetCredentials initializes StorageManager with the provided AWS credentials
func (m *StorageManager) SetCredentials(accessKeyID, secretAccessKey string) {
	m.creds = credentials.NewStaticCredentials(accessKeyID, secretAccessKey, "")
}

func (m *StorageManager) logf(s string, args ...interface{}) {
	if m.LogFunc != nil {
		m.LogFunc(s, args...)
	}
}

type awsLogger struct {
	lf func(string, ...interface{})
}

func (al *awsLogger) Log(args ...interface{}) {
	if al.lf != nil {
		al.lf("", args...)
	}
}

func (m *StorageManager) getSession(region string, retries uint) *session.Session {
	return session.New(
		aws.NewConfig().WithRegion(region).WithMaxRetries(int(retries)).WithCredentials(m.creds).WithLogger(&awsLogger{lf: m.LogFunc}))
}

// Push reads data from in, pushes it to S3 and returns the S3 location URL (optionally presigned)
func (m *StorageManager) Push(contentType string, in io.Reader, opts Options) (string, error) {
	loc, err := m.pushdata(contentType, in, opts, s3.BucketCannedACLPrivate)
	if err != nil {
		return "", err
	}
	if opts.PresignTTLMinutes > 0 {
		loc, err = m.presignedURL(opts)
		if err != nil {
			return "", err
		}
	}
	return loc, nil
}

func (m *StorageManager) pushdata(contentType string, in io.Reader, opts Options, perms string) (string, error) {
	sess := m.getSession(opts.Region, opts.MaxRetries)
	u := s3manager.NewUploaderWithClient(s3.New(sess), func(u *s3manager.Uploader) {
		u.Concurrency = int(opts.Concurrency)
	})
	ct := contentType
	var p *string
	if perms != "" {
		p = &perms
	}
	ui := &s3manager.UploadInput{
		ContentType: &ct,
		Bucket:      &opts.Bucket,
		Body:        in,
		Key:         &opts.Key,
		ACL:         p,
	}
	uo, err := u.Upload(ui)
	if err != nil {
		return "", err
	}
	m.logf("S3 write location: %v", uo.Location)
	m.logf("S3 version ID: %v", uo.VersionID)
	m.logf("S3 upload ID: %v", uo.UploadID)
	return uo.Location, nil
}

func (m *StorageManager) presignedURL(opts Options) (string, error) {
	sess := m.getSession(opts.Region, opts.MaxRetries)
	svc := s3.New(sess)
	req, _ := svc.GetObjectRequest(&s3.GetObjectInput{
		Bucket: &opts.Bucket,
		Key:    &opts.Key,
	})

	return req.Presign(time.Duration(opts.PresignTTLMinutes) * time.Minute)
}
