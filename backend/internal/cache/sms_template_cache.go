/**
 * @Author: Nan
 * @Date: 2025/2/8 09:50
 */

package cache

import (
	"backend/model"
)

type SmsTemplateCache struct {
	*BasicCache[model.SmsTemplate]
}

const SmsTemplateCacheKey = "SMS_TEMPLATE_CACHE_KEY_" // 短信模板缓存key

func NewSmsTemplateCache(cacher Cacher) *SmsTemplateCache {
	return &SmsTemplateCache{BasicCache: NewBasicCache[model.SmsTemplate](cacher, SmsTemplateCacheKey)}
}

// SetSmsTemplateList 传入短信模版列表缓存短信模版
func (s *SmsTemplateCache) SetSmsTemplateList(smsTemplateList []model.SmsTemplate) {
	for _, temp := range smsTemplateList {
		s.SetExpiration(temp.TemplateCode, temp, 60*60*24*30) // 缓存时间为30天
	}
}

// Set 传入短信模版缓存短信模版30天
func (s *SmsTemplateCache) Set(key string, smsTemplate model.SmsTemplate) error {
	err := s.SetExpiration(key, smsTemplate, 60*60*24*30) // 缓存时间为30天
	if err != nil {
		return err
	}
	return nil
}
