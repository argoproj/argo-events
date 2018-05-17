/*
Copyright 2018 BlackRock, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package store

import (
	"io/ioutil"

	"github.com/blackrock/axis/pkg/apis/sensor/v1alpha1"
	minio "github.com/minio/minio-go"
)

// S3Reader implements the ArtifactReader interface and allows reading artifacts from S3 compatible API store
type S3Reader struct {
	client *minio.Client
	s3     *v1alpha1.S3Artifact
	creds  *Credentials
}

// NewS3Reader creates a new ArtifactReader for an S3 compatible store
func NewS3Reader(s3 *v1alpha1.S3Artifact, creds *Credentials) (ArtifactReader, error) {
	client, err := newMinioClient(s3, *creds)
	if err != nil {
		return nil, err
	}
	return &S3Reader{
		client: client,
		s3:     s3,
		creds:  creds,
	}, nil
}

func (reader *S3Reader) Read() ([]byte, error) {
	obj, err := reader.client.GetObject(reader.s3.Bucket, reader.s3.Key, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	defer obj.Close()
	b, err := ioutil.ReadAll(obj)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// newMinioClient instantiates a new minio client object to access s3 compatible APIs
func newMinioClient(s3 *v1alpha1.S3Artifact, creds Credentials) (*minio.Client, error) {
	var minioClient *minio.Client
	var err error
	if s3.Region != "" {
		minioClient, err = minio.NewWithRegion(s3.Endpoint, creds.accessKey, creds.secretKey, !s3.Insecure, s3.Region)
	} else {
		minioClient, err = minio.New(s3.Endpoint, creds.accessKey, creds.secretKey, !s3.Insecure)
	}
	if err != nil {
		return nil, err
	}
	return minioClient, nil
}
