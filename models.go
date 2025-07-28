package photofunia

type photoFuniaResponse struct {
	Response struct {
		Key      string `json:"key"`
		Server   int    `json:"server"`
		Existed  bool   `json:"existed"`
		Expiry   int64  `json:"expiry"`
		Created  int64  `json:"created"`
		Lifetime int    `json:"lifetime"`
		Image    struct {
			Highres struct {
				URL    string `json:"url"`
				Width  int    `json:"width"`
				Height int    `json:"height"`
			} `json:"highres"`
			Preview struct {
				URL    string `json:"url"`
				Width  int    `json:"width"`
				Height int    `json:"height"`
			} `json:"preview"`
			Thumb struct {
				URL    string `json:"url"`
				Width  int    `json:"width"`
				Height int    `json:"height"`
			} `json:"thumb"`
		} `json:"image"`
		Sid string `json:"sid"`
	} `json:"response"`
}
