package addons

import (
	"encoding/json"
	"io"
	"net/http"
)

type Desk struct {
	ID     string `json:"id"`
	Height int    `json:"value"`
}

func GetDeskHeight(deskurl string) (int, error) {
	resp, err := http.Get(deskurl + "/sensor/desk_height")
	d := Desk{}
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}
	err = json.Unmarshal(body, &d)
	if err != nil {
		return 0, err
	}

	return d.Height, nil
}
