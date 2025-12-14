package services

import (
	"fmt"
	"reflect"

	"github.com/atticus6/echPlus/apps/desktop/config"
)

func MergeStructs(dst, src any) {
	dstVal := reflect.ValueOf(dst).Elem()
	srcVal := reflect.ValueOf(src).Elem()

	for i := 0; i < srcVal.NumField(); i++ {
		srcField := srcVal.Field(i)
		fieldName := srcVal.Type().Field(i).Name
		dstField := dstVal.FieldByName(fieldName)

		// 只合并非零值
		if dstField.IsValid() && !srcField.IsZero() {
			dstField.Set(srcField)
		}
	}
}

type ConfigService struct{}

func (c *ConfigService) GetValue() config.ConfigType {
	return config.ConfigState
}

func (c *ConfigService) ChangeValue(v config.ConfigType) {
	MergeStructs(&config.ConfigState, &v)
	if v.SelectNodeId != 0 {
		ProxyServerInstance.SwitchNode(v.SelectNodeId)
	} else {
		origonCfg := s.GetConfig()
		v2 := config.ConfigState.GetproxyConfig()
		MergeStructs(&origonCfg, &v2)
		s.UpdateConfig(origonCfg)
	}

	fmt.Println(config.ConfigState, config.ConfigState)
}
