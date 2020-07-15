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

package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/store"
)

// GetAWSCredFromEnvironment reads credential stored in ENV by using envFrom.
func GetAWSCredFromEnvironment(access *corev1.SecretKeySelector, secret *corev1.SecretKeySelector) (*credentials.Credentials, error) {
	accessKey, ok := common.GetEnvFromSecret(access)
	if !ok {
		return nil, errors.Errorf("can not find envFrom %v", access)
	}
	secretKey, ok := common.GetEnvFromSecret(secret)
	if !ok {
		return nil, errors.Errorf("can not find envFrom %v", secret)
	}
	return credentials.NewStaticCredentialsFromCreds(credentials.Value{
		AccessKeyID:     accessKey,
		SecretAccessKey: secretKey,
	}), nil
}

// GetAWSCreds reads credential stored in Kubernetes secrets and return it.
func GetAWSCreds(client kubernetes.Interface, namespace string, access *corev1.SecretKeySelector, secret *corev1.SecretKeySelector) (*credentials.Credentials, error) {
	accessKey, err := store.GetSecrets(client, namespace, access.Name, access.Key)
	if err != nil {
		return nil, err
	}
	secretKey, err := store.GetSecrets(client, namespace, secret.Name, secret.Key)
	if err != nil {
		return nil, err
	}

	return credentials.NewStaticCredentialsFromCreds(credentials.Value{
		AccessKeyID:     accessKey,
		SecretAccessKey: secretKey,
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

// CreateAWSSessionWithCredsInEnv based on credentials in ENV	 return a aws session
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

// CreateAWSSession based on credentials settings return a aws session
func CreateAWSSession(client kubernetes.Interface, namespace, region string, roleARN string, accessKey *corev1.SecretKeySelector, secretKey *corev1.SecretKeySelector) (*session.Session, error) {
	if roleARN != "" {
		return GetAWSAssumeRoleCreds(roleARN, region)
	}

	if accessKey == nil && secretKey == nil {
		return GetAWSSessionWithoutCreds(region)
	}

	creds, err := GetAWSCreds(client, namespace, accessKey, secretKey)
	if err != nil {
		return nil, err
	}

	return GetAWSSession(creds, region)
}
