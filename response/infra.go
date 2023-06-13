package response

type Infra struct {
	Code    int32       `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

type PageQuery struct {
	PageEnable    bool   `json:"page_enable"`
	Total         int64  `json:"total"`
	Page          int    `json:"page"`
	PageSize      int    `json:"page_size"`
	SortDirection string `json:"-"`
}

type DataWithPagenation struct {
	Data interface{} `json:"data"`
	Page PageQuery   `json:"page"`
}
