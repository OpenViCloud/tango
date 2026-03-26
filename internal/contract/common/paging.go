package common

// BaseRequestModel matches the shared pagination/sort contract used by clients.
type BaseRequestModel struct {
	PageIndex  int    `json:"pageIndex" form:"pageIndex"`
	PageSize   int    `json:"pageSize" form:"pageSize"`
	SearchText string `json:"searchText" form:"searchText"`
	OrderBy    string `json:"orderBy" form:"orderBy"`
	Ascending  bool   `json:"ascending" form:"ascending"`
}

// BaseResponseModel wraps paginated list results returned by application queries.
type BaseResponseModel[T any] struct {
	Items      []T   `json:"items"`
	PageIndex  int   `json:"pageIndex"`
	PageSize   int   `json:"pageSize"`
	TotalItems int64 `json:"totalItems"`
	TotalPage  int   `json:"totalPage"`
}
