/**
 * @Author: Nan
 * @Date: 2026/2/17
 */

package storage

import (
	"backend/config"
	"fmt"
)

// NewStorage 根据配置创建对应的存储实现
func NewStorage(cfg *config.Server) (Storage, error) {
	driver := cfg.Upload.Driver
	if driver == "" {
		driver = "local"
	}

	switch driver {
	case "local":
		return NewLocalStorage(cfg.Upload.Dir, cfg.Upload.BaseURL), nil
	case "oss":
		ossCfg := cfg.Upload.OSS
		return NewOSSStorage(OSSConfig{
			Endpoint:        ossCfg.Endpoint,
			AccessKeyID:     ossCfg.AccessKeyID,
			AccessKeySecret: ossCfg.AccessKeySecret,
			BucketName:      ossCfg.BucketName,
			BaseURL:         ossCfg.BaseURL,
			BasePath:        ossCfg.BasePath,
		})
	default:
		return nil, fmt.Errorf("不支持的存储类型: %s", driver)
	}
}
