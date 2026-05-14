package util

// Paginate 校验分页参数并返回 offset，统一分页逻辑
func Paginate(page, pageSize int) (offset, validPage, validSize int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return (page - 1) * pageSize, page, pageSize
}
