package tencent

import (
	"fmt"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	sms "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/sms/v20210111"
)

type SMSClient struct {
	client     *sms.Client
	appID      string
	signName   string
	templateID string
}

// NewSMSClient 创建短信客户端（支持永久密钥）
func NewSMSClient(secretID, secretKey, appID, signName, templateID string) (*SMSClient, error) {
	credential := common.NewCredential(secretID, secretKey)
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "sms.tencentcloudapi.com"

	client, err := sms.NewClient(credential, "ap-beijing", cpf)
	if err != nil {
		return nil, fmt.Errorf("创建SMS客户端失败: %w", err)
	}

	return &SMSClient{
		client:     client,
		appID:      appID,
		signName:   signName,
		templateID: templateID,
	}, nil
}

// NewSMSClientWithToken 创建短信客户端（支持 STS 临时密钥）
func NewSMSClientWithToken(secretID, secretKey, token, appID, signName, templateID string) (*SMSClient, error) {
	credential := common.NewTokenCredential(secretID, secretKey, token)
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "sms.tencentcloudapi.com"

	client, err := sms.NewClient(credential, "ap-beijing", cpf)
	if err != nil {
		return nil, fmt.Errorf("创建SMS客户端失败: %w", err)
	}

	return &SMSClient{
		client:     client,
		appID:      appID,
		signName:   signName,
		templateID: templateID,
	}, nil
}

func (s *SMSClient) SendVerifyCode(phone, code string) error {
	request := sms.NewSendSmsRequest()
	request.SmsSdkAppId = common.StringPtr(s.appID)
	request.SignName = common.StringPtr(s.signName)
	request.TemplateId = common.StringPtr(s.templateID)
	request.PhoneNumberSet = common.StringPtrs([]string{"+86" + phone})
	request.TemplateParamSet = common.StringPtrs([]string{code}) // 模板参数：验证码

	response, err := s.client.SendSms(request)
	if err != nil {
		return fmt.Errorf("发送短信失败: %w", err)
	}

	if response.Response != nil && len(response.Response.SendStatusSet) > 0 {
		status := response.Response.SendStatusSet[0]
		if *status.Code != "Ok" {
			return fmt.Errorf("发送短信失败: %s", *status.Message)
		}
	}

	return nil
}
