package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
)

type versionConfig struct {
	Version *string `json:"version"`
	Build   *int64  `json:"build"`
}

var canonicalVersion = regexp.MustCompile("^(0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)$")

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "Lỗi phiên bản: "+format+"\n", args...)
	os.Exit(1)
}

func main() {
	configPath := os.Getenv("TRUYENTEP_VERSION_CONFIG")
	if configPath == "" {
		fail("không nhận được đường dẫn version config.")
	}

	configFile, err := os.Open(configPath)
	if err != nil {
		fail("không đọc được file config: %v", err)
	}
	defer configFile.Close()

	decoder := json.NewDecoder(configFile)
	decoder.DisallowUnknownFields()

	var config versionConfig
	if err := decoder.Decode(&config); err != nil {
		fail("JSON không hợp lệ: %v", err)
	}

	var extra json.RawMessage
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			fail("config chỉ được chứa đúng một JSON object.")
		}
		fail("JSON có dữ liệu thừa hoặc không hợp lệ: %v", err)
	}

	if config.Version == nil {
		fail("thiếu trường version dạng chuỗi.")
	}
	if config.Build == nil {
		fail("thiếu trường build dạng số nguyên.")
	}
	if !canonicalVersion.MatchString(*config.Version) {
		fail("version phải gồm đúng ba số nguyên không có số 0 thừa: X.Y.Z.")
	}
	if *config.Build < 1 || *config.Build > 9999 {
		fail("build phải là số nguyên dương từ 1 đến 9999.")
	}

	fmt.Printf("%s\t%d\n", *config.Version, *config.Build)
}
