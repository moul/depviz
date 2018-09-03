package main

import (
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var vph viperPFlagHelper

// based on https://github.com/spf13/viper/issues/82
type viperPFlagBinding struct {
	configName string
	flagValue  pflag.Value
}
type viperPFlagHelper struct {
	bindings []viperPFlagBinding
}

func (vph *viperPFlagHelper) BindPFlag(configName string, flag *pflag.Flag) (err error) {
	err = viper.BindPFlag(configName, flag)
	if err == nil {
		vph.bindings = append(vph.bindings, viperPFlagBinding{configName, flag.Value})
	}
	return
}
func (vph *viperPFlagHelper) setPFlagsFromViper() {
	for _, v := range vph.bindings {
		if v.flagValue.String() != "" {
			v.flagValue.Set(viper.GetString(v.configName))
		}
	}
}
func (vph *viperPFlagHelper) BindPFlags(flags *pflag.FlagSet) (err error) {
	flags.VisitAll(func(flag *pflag.Flag) {
		if err = vph.BindPFlag(flag.Name, flag); err != nil {
			return
		}
	})
	return nil
}
