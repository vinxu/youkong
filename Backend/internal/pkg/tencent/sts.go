package tencent

import (
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	sts "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/sts/v20180813"
)

// STSCredentials STS 临时凭证
type STSCredentials struct {
	TmpSecretID  string `json:"accessKeyId"`
	TmpSecretKey string `json:"secretAccessKey"`
	SessionToken string `json:"sessionToken"`
	Bucket       string `json:"bucket"`
	Region       string `json:"region"`
	Prefix       string `json:"prefix"`
}

// GenerateSTSCredentials 生成 COS 临时上传凭证
func GenerateSTSCredentials(secretID, secretKey, bucket, region, prefix string) (*STSCredentials, error) {
	credential := common.NewCredential(secretID, secretKey)
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "sts.tencentcloudapi.com"

	client, err := sts.NewClient(credential, region, cpf)
	if err != nil {
		return nil, err
	}

	// 构建 COS 上传策略
	resource := "qcs::cos:" + region + ":uid/1323065328:" + bucket + "/" + prefix + "*"

	policy := `{
		"version": "2.0",
		"statement": [{
			"effect": "allow",
			"action": [
				"name/cos:PutObject",
				"name/cos:PostObject",
				"name/cos:PutObjectACL"
			],
			"resource": ["` + resource + `"]
		}]
	}`

	request := sts.NewGetFederationTokenRequest()
	name := "youkong-gif-upload"
	request.Name = &name
	durationSeconds := uint64(1800) // 30 分钟
	request.DurationSeconds = &durationSeconds
	request.Policy = &policy

	response, err := client.GetFederationToken(request)
	if err != nil {
		return nil, err
	}

	cred := response.Response.Credentials
	return &STSCredentials{
		TmpSecretID:  *cred.TmpSecretId,
		TmpSecretKey: *cred.TmpSecretKey,
		SessionToken: *cred.Token,
		Bucket:       bucket,
		Region:       region,
		Prefix:       prefix,
	}, nil
}
