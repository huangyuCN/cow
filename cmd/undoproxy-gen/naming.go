package main

// singular 将字段名复数形式转为单数（初版：末尾 s 去掉）。
func singular(field string) string {
	if len(field) > 1 && field[len(field)-1] == 's' {
		return field[:len(field)-1]
	}
	return field
}

// sliceMethods 字段级 slice 的方法名。
type sliceMethods struct {
	Append, SetAt, RemoveAt, Truncate string
}

func sliceMethodNames(field string) sliceMethods {
	return sliceMethods{
		Append:   "Append" + field,
		SetAt:    "Set" + field + "At",
		RemoveAt: "Remove" + field + "At",
		Truncate: "Truncate" + field,
	}
}

// mapForWriteName 内层 map 的 Get*MapForWrite 方法名。
func mapForWriteName(field string) string {
	return "Get" + field + "MapForWrite"
}

// elemAtForWriteName map[k][i] 元素 *Struct 的 Get 方法名。
func elemAtForWriteName(elemTypeName string) string {
	return "Get" + elemTypeName + "AtForWrite"
}
