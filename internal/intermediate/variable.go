package intermediate

import (
	"fmt"
	"strconv"
)

type Variable struct {
	Name         string
	AccessType   AccessType
	ProseType    ProseType
	DefaultValue interface{}
}

func NewVariable(name, atype, ptype string) (*Variable, error) {
	v := &Variable{Name: name}

	v.AccessType = GetAccessType(atype)
	if v.AccessType == AccesSTypeInvalid {
		return nil, fmt.Errorf("invalid access type for prose variable: %s %s", name, atype)
	}

	v.ProseType = GetProseType(ptype)
	if v.ProseType == ProseTypeInvalid {
		return nil, fmt.Errorf("invalid type for prose variable: %s %s", name, ptype)
	}

	v.SetDefaultValue()

	return v, nil
}

func (v *Variable) SetDefaultValue() {
	switch v.ProseType {
	case ProseTypeInt:
		v.DefaultValue = int64(0)
	case ProseTypeBool:
		v.DefaultValue = false
	case ProseTypeString:
		v.DefaultValue = ""
	}
}

func (v *Variable) SetValue(s string) error {
	switch v.ProseType {
	case ProseTypeInt:
		val, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return err
		}
		v.DefaultValue = val
	case ProseTypeBool:
		val, err := strconv.ParseBool(s)
		if err != nil {
			return err
		}
		v.DefaultValue = val
	case ProseTypeString:
		v.DefaultValue = s
	}
	return nil
}
