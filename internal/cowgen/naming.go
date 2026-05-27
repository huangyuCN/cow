package cowgen

// Singular 将字段名复数形式转为单数（当前实现：末尾 s 去掉）。
func Singular(field string) string {
	if len(field) > 1 && field[len(field)-1] == 's' {
		return field[:len(field)-1]
	}
	return field
}

// SliceMethods 字段级 slice 的方法名。
type SliceMethods struct {
	Append, SetAt, RemoveAt, Truncate string
}

// SliceMethodNames 返回 slice 相关代理方法名。
func SliceMethodNames(field string) SliceMethods {
	return SliceMethods{
		Append:   "Append" + field,
		SetAt:    "Set" + field + "At",
		RemoveAt: "Remove" + field + "At",
		Truncate: "Truncate" + field,
	}
}

// MapForWriteName 内层 map 的 Get*MapForWrite 方法名。
func MapForWriteName(field string) string {
	return "Get" + field + "MapForWrite"
}

// ElemAtForWriteName map[k][i] 元素 *Struct 的 Get 方法名。
func ElemAtForWriteName(elemTypeName string) string {
	return "Get" + elemTypeName + "AtForWrite"
}

// PtrGetForWriteName 指针字段 Get 方法名（MainHero → GetMainHeroForWrite）。
func PtrGetForWriteName(field string) string {
	return "Get" + field + "ForWrite"
}

// MapKeyGetForWriteName map[k]*Struct 的 Get 方法名（Heros → GetHeroForWrite）。
func MapKeyGetForWriteName(singular string) string {
	return "Get" + singular + "ForWrite"
}

// PutFieldName 标量/map Put 方法名。
func PutFieldName(field string) string {
	return "Put" + field
}

// PtrSetName 指针字段整槽替换方法名（MainHero → SetMainHero）。
func PtrSetName(field string) string {
	return "Set" + field
}

// MapRemoveName map 删 key 方法名（Heros → RemoveHeros）。
func MapRemoveName(field string) string {
	return "Remove" + field
}
