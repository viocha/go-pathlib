package common

import (
	"errors"
	"fmt"
)

// 使用parent错误包装一个子错误，并添加格式信息描述父错误
func WrapSub(parent error, err error, format string, a ...any) error {
	return errors.Join(
		WrapMsg(parent, format, a...),
		err,
	)
}

// 用错误包装一个具体消息
func WrapMsg(err error, format string, a ...any) error {
	return errors.Join(
		err,
		fmt.Errorf(format, a...),
	)
}

func ParseOptional[T any](arg []T, defaultValue T) T {
	if len(arg) == 0 {
		return defaultValue
	}
	return arg[0]
}
