package dto

type StatsCountItem struct {
	Date  string `json:"date"`
	Count int64  `json:"count"`
}

type DeviceStatItem struct {
	DeviceType string `json:"device_type"`
	Count      int64  `json:"count"`
}

type RefererStatItem struct {
	Refer string `json:"refer"`
	Count int64  `json:"count"`
}

type StatsResponse struct {
	TotalClicks    int64             `json:"total_clicks"`
	UniqueVisitors int64             `json:"unique_visitors"`
	DailyClicks    []StatsCountItem  `json:"daily_clicks"`
	DeviceStats    []DeviceStatItem  `json:"device_stats"`
	Referers       []RefererStatItem `json:"referers"`
}

type AccessLogItem struct {
	AccessTime   string `json:"access_time"`
	IP           string `json:"ip"`
	DeviceType   string `json:"device_type"`
	Referer      string `json:"referer"`
	Browser      string `json:"browser"`
	Status       int8   `json:"status"`
	ErrorMessage string `json:"error_message"`
}

type LogListResponse struct {
	Total      int64           `json:"total"`
	Page       int             `json:"page"`
	PageSize   int             `json:"page_size"`
	TotalPages int             `json:"total_pages"`
	List       []AccessLogItem `json:"list"`
}
