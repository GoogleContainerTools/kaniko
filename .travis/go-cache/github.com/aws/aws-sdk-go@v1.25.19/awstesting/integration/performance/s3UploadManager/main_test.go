// +build integration,perftest

package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/aws/aws-sdk-go/awstesting/integration"
	"github.com/aws/aws-sdk-go/internal/sdkio"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

var benchConfig BenchmarkConfig

type BenchmarkConfig struct {
	bucket       string
	tempdir      string
	clientConfig ClientConfig
}

func (b *BenchmarkConfig) SetupFlags(prefix string, flagSet *flag.FlagSet) {
	flagSet.StringVar(&b.bucket, "bucket", "", "Bucket to use for benchmark")
	flagSet.StringVar(&b.tempdir, "temp", os.TempDir(), "location to create temporary files")
	b.clientConfig.SetupFlags(prefix, flagSet)
}

var benchStrategies = []struct {
	name           string
	bufferProvider s3manager.ReadSeekerWriteToProvider
}{
	{name: "Unbuffered", bufferProvider: nil},
	{name: "Buffered", bufferProvider: s3manager.NewBufferedReadSeekerWriteToPool(1024 * 1024)},
}

func BenchmarkInMemory(b *testing.B) {
	memBreader := bytes.NewReader(make([]byte, 1*1024*1024*1024))

	baseSdkConfig := SDKConfig{WithUnsignedPayload: true, ExpectContinue: true, WithContentMD5: false}

	key := integration.UniqueID()
	// Concurrency: 5, 10, 100
	for _, concurrency := range []int{s3manager.DefaultUploadConcurrency, 2 * s3manager.DefaultUploadConcurrency, 100} {
		b.Run(fmt.Sprintf("%d_Concurrency", concurrency), func(b *testing.B) {
			// PartSize: 5 MB, 25 MB, 100 MB
			for _, partSize := range []int64{s3manager.DefaultUploadPartSize, 25 * 1024 * 1024, 100 * 1024 * 1024} {
				b.Run(fmt.Sprintf("%s_PartSize", integration.SizeToName(int(partSize))), func(b *testing.B) {
					sdkConfig := baseSdkConfig

					sdkConfig.BufferProvider = nil
					sdkConfig.Concurrency = concurrency
					sdkConfig.PartSize = partSize

					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						benchUpload(b, benchConfig.bucket, key, memBreader, sdkConfig, benchConfig.clientConfig)
						_, err := memBreader.Seek(0, sdkio.SeekStart)
						if err != nil {
							b.Fatalf("failed to seek to start of file: %v", err)
						}
					}
				})
			}
		})
	}
}

func BenchmarkUpload(b *testing.B) {
	baseSdkConfig := SDKConfig{WithUnsignedPayload: true, ExpectContinue: true, WithContentMD5: false}

	// FileSizes: 5 MB, 1 GB, 10 GB
	for _, fileSize := range []int64{5 * 1024 * 1024, 1024 * 1024 * 1024, 10 * 1024 * 1024 * 1024} {
		b.Run(fmt.Sprintf("%s_File", integration.SizeToName(int(fileSize))), func(b *testing.B) {
			b.Logf("creating file of size: %s", integration.SizeToName(int(fileSize)))
			file, err := integration.CreateFileOfSize(benchConfig.tempdir, fileSize)
			if err != nil {
				b.Fatalf("failed to create file: %v", err)
			}

			// Concurrency: 5, 10, 100
			for _, concurrency := range []int{s3manager.DefaultUploadConcurrency, 2 * s3manager.DefaultUploadConcurrency, 100} {
				b.Run(fmt.Sprintf("%d_Concurrency", concurrency), func(b *testing.B) {
					// PartSize: 5 MB, 25 MB, 100 MB
					for _, partSize := range []int64{s3manager.DefaultUploadPartSize, 25 * 1024 * 1024, 100 * 1024 * 1024} {
						if partSize > fileSize {
							continue
						}
						b.Run(fmt.Sprintf("%s_PartSize", integration.SizeToName(int(partSize))), func(b *testing.B) {
							for _, strat := range benchStrategies {
								b.Run(strat.name, func(b *testing.B) {
									sdkConfig := baseSdkConfig

									sdkConfig.BufferProvider = strat.bufferProvider
									sdkConfig.Concurrency = concurrency
									sdkConfig.PartSize = partSize

									b.ResetTimer()
									for i := 0; i < b.N; i++ {
										benchUpload(b, benchConfig.bucket, filepath.Base(file.Name()), file, sdkConfig, benchConfig.clientConfig)
										_, err := file.Seek(0, sdkio.SeekStart)
										if err != nil {
											b.Fatalf("failed to seek to start of file: %v", err)
										}
									}
								})
							}
						})
					}
				})
			}

			os.Remove(file.Name())
			file.Close()
		})
	}
}

func benchUpload(b *testing.B, bucket, key string, reader io.ReadSeeker, sdkConfig SDKConfig, clientConfig ClientConfig) {
	uploader := newUploader(clientConfig, sdkConfig, SetUnsignedPayload)
	_, err := uploader.Upload(&s3manager.UploadInput{
		Bucket: &bucket,
		Key:    &key,
		Body:   reader,
	})
	if err != nil {
		b.Fatalf("failed to upload object, %v", err)
	}
}

func TestMain(m *testing.M) {
	benchConfig.SetupFlags("", flag.CommandLine)
	flag.Parse()
	os.Exit(m.Run())
}
