/**
 * @Author: Nan
 * @Date: 2024/6/7 下午10:54
 */

package initialize

import (
	"backend/internal/utils"

	"github.com/gin-gonic/gin/binding"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
)

func InitValidators() (map[string]ut.Translator, error) {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		return utils.InitializeValidator(v)
	}
	return map[string]ut.Translator{}, nil
}
