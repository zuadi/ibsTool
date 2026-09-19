package models

type Info struct {
	Name        string `json:"name"`
	Author      string `json:"author"`
	Version     string `json:"version"`
	ReleaseDate string `json:"releaseDate"`
}

func GetInfo() Info {
	return Info{
		Name:        "IBS Server",
		Author:      "Adrian Zuercher",
		Version:     "0.0.1",
		ReleaseDate: "10.07.2026",
	}
}
