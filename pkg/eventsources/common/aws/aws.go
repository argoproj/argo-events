/*
Copyright 2018 The Argoproj Authors.

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

package aws

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/credentials"
	"github.com/aws/aws-sdk-go-v2/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/aws/session"
	corev1 "k8s.io/api/core/v1"

	sharedutil "github.com/argoproj/argo-events/pkg/shared/util"
)

// GetAWSCredFromEnvironment reads credential stored in ENV by using envFrom.
func GetAWSCredFromEnvironment(access *corev1.SecretKeySelector, secret *corev1.SecretKeySelector) (*credentials.Credentials, error) {
	accessKey, ok := sharedutil.GetEnvFromSecret(access)
	if !ok {
		return nil, fmt.Errorf("can not find envFrom %v", access)
	}
	secretKey, ok := sharedutil.GetEnvFromSecret(secret)
	if !ok {
		return nil, fmt.Errorf("can not find envFrom %v", secret)
	}
	return credentials.NewStaticCredentialsFromCreds(credentials.Value{
		AccessKeyID:     accessKey,
		SecretAccessKey: secretKey,
	}), nil
}

// GetAWSCredFromVolume reads credential stored in mounted secret volume.
func GetAWSCredFromVolume(access *corev1.SecretKeySelector, secret *corev1.SecretKeySelector, sessionToken *corev1.SecretKeySelector) (*credentials.Credentials, error) {
	accessKey, err := sharedutil.GetSecretFromVolume(access)
	if err != nil {
		return nil, fmt.Errorf("can not find access key, %w", err)
	}
	secretKey, err := sharedutil.GetSecretFromVolume(secret)
	if err != nil {
		return nil, fmt.Errorf("can not find secret key, %w", err)
	}

	var token string
	if sessionToken != nil {
		token, err = sharedutil.GetSecretFromVolume(sessionToken)
		if err != nil {
			return nil, fmt.Errorf("can not find session token, %w", err)
		}
	}

	return credentials.NewStaticCredentialsFromCreds(credentials.Value{
		AccessKeyID:     accessKey,
		SecretAccessKey: secretKey,
		SessionToken:    token,
	}), nil
}

func GetAWSSession(creds *credentials.Credentials, region string) (*session.Session, error) {
	return session.NewSession(&aws.Config{
		Region:      &region,
		Credentials: creds,
	})
}

func GetAWSSessionWithoutCreds(region string) (*session.Session, error) {
	return session.NewSession(&aws.Config{
		Region: &region,
	})
}

func GetAWSAssumeRoleCreds(roleARN, region string) (*session.Session, error) {
	sess := session.Must(session.NewSession())
	creds := stscreds.NewCredentials(sess, roleARN)
	return GetAWSSession(creds, region)
}

// CreateAWSSessionWithCredsInEnv based on credentials in ENV, return a aws session
func CreateAWSSessionWithCredsInEnv(region string, roleARN string, accessKey *corev1.SecretKeySelector, secretKey *corev1.SecretKeySelector) (*session.Session, error) {
	if roleARN != "" {
		return GetAWSAssumeRoleCreds(roleARN, region)
	}

	if accessKey == nil && secretKey == nil {
		return GetAWSSessionWithoutCreds(region)
	}

	creds, err := GetAWSCredFromEnvironment(accessKey, secretKey)
	if err != nil {
		return nil, err
	}

	return GetAWSSession(creds, region)
}

// CreateAWSSessionWithCredsInVolume based on credentials in mounted volumes, return a aws session
func CreateAWSSessionWithCredsInVolume(region string, roleARN string, accessKey *corev1.SecretKeySelector, secretKey *corev1.SecretKeySelector, sessionToken *corev1.SecretKeySelector) (*session.Session, error) {
	if roleARN != "" {
		return GetAWSAssumeRoleCreds(roleARN, region)
	}

	if accessKey == nil && secretKey == nil {
		return GetAWSSessionWithoutCreds(region)
	}

	creds, err := GetAWSCredFromVolume(accessKey, secretKey, sessionToken)
	if err != nil {
		return nil, err
	}

	return GetAWSSession(creds, region)
}
