package helpers

import "reflect"

func GetType(myvar interface{}) string {
	return reflect.TypeOf(myvar).String()
}
