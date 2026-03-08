package middleware

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"git.neolidy.top/neo/storybook/internal/api"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// BindJSON 统一处理 JSON 请求绑定与参数校验错误。
func BindJSON(c *gin.Context, obj any) bool {
	if err := c.ShouldBindJSON(obj); err != nil {
		api.BadRequest(c, "参数验证失败", parseBindErrors(obj, err)...)
		c.Abort()
		return false
	}
	return true
}

func parseBindErrors(obj any, err error) []api.ErrorItem {
	if err == nil {
		return nil
	}

	var vErrs validator.ValidationErrors
	if errors.As(err, &vErrs) {
		items := make([]api.ErrorItem, 0, len(vErrs))
		for _, fe := range vErrs {
			field := resolveJSONField(obj, fe.Field())
			items = append(items, api.ErrorItem{Field: field, Message: validationMessage(fe)})
		}
		return items
	}

	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &typeErr) {
		field := strings.TrimSpace(typeErr.Field)
		if field == "" {
			field = "body"
		}
		return []api.ErrorItem{{Field: field, Message: "字段类型不匹配"}}
	}

	var syntaxErr *json.SyntaxError
	if errors.As(err, &syntaxErr) {
		return []api.ErrorItem{{Field: "body", Message: "JSON格式错误"}}
	}

	msg := strings.TrimSpace(err.Error())
	if msg == "" {
		msg = "请求参数不合法"
	}
	return []api.ErrorItem{{Field: "body", Message: msg}}
}

func resolveJSONField(obj any, fieldName string) string {
	if obj == nil || fieldName == "" {
		return strings.ToLower(fieldName)
	}

	t := reflect.TypeOf(obj)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return strings.ToLower(fieldName)
	}

	if sf, ok := t.FieldByName(fieldName); ok {
		tag := sf.Tag.Get("json")
		if tag != "" {
			parts := strings.Split(tag, ",")
			if len(parts) > 0 && parts[0] != "" && parts[0] != "-" {
				return parts[0]
			}
		}
	}

	return strings.ToLower(fieldName)
}

func validationMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "不能为空"
	case "email":
		return "邮箱格式不正确"
	case "oneof":
		return "取值不合法"
	case "min":
		if fe.Kind() == reflect.String || fe.Kind() == reflect.Slice || fe.Kind() == reflect.Array {
			return fmt.Sprintf("长度至少为%s", fe.Param())
		}
		return fmt.Sprintf("最小值为%s", fe.Param())
	case "max":
		if fe.Kind() == reflect.String || fe.Kind() == reflect.Slice || fe.Kind() == reflect.Array {
			return fmt.Sprintf("长度不能超过%s", fe.Param())
		}
		return fmt.Sprintf("最大值为%s", fe.Param())
	case "gt":
		return fmt.Sprintf("必须大于%s", fe.Param())
	case "gte":
		return fmt.Sprintf("必须大于等于%s", fe.Param())
	case "lt":
		return fmt.Sprintf("必须小于%s", fe.Param())
	case "lte":
		return fmt.Sprintf("必须小于等于%s", fe.Param())
	default:
		if fe.Param() != "" {
			if _, err := strconv.Atoi(fe.Param()); err == nil {
				return "参数不合法"
			}
		}
		return "参数不合法"
	}
}
