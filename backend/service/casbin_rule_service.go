/**
 * @Author: Nan
 * @Date: 2024/6/22 下午2:34
 */

package service

import "backend/repository"

type CasbinRuleService struct {
	casbinRuleRepo repository.CasbinRuleRepository
}

func NewCasbinRuleService(casbinRuleRepo repository.CasbinRuleRepository) *CasbinRuleService {
	return &CasbinRuleService{casbinRuleRepo: casbinRuleRepo}
}

func (s *CasbinRuleService) AddPolicy(role, path, method string) (bool, error) {
	return s.casbinRuleRepo.AddPolicy(role, path, method)
}

func (s *CasbinRuleService) RemovePolicy(role, path, method string) (bool, error) {
	return s.casbinRuleRepo.RemovePolicy(role, path, method)
}

func (s *CasbinRuleService) RemoveFilteredPolicy(fieldIndex int, fieldValues ...string) (bool, error) {
	return s.casbinRuleRepo.RemoveFilteredPolicy(fieldIndex, fieldValues...)
}
